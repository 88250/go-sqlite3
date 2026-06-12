//go:build sqlite_fts5 || fts5
// +build sqlite_fts5 fts5

package sqlite3

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
)

// 基准语料：默认用内置混合繁简文本平铺生成（确定性、自包含、无外部依赖）；
// 设置环境变量 SIYUAN_BENCH_CORPUS=<纯文本文件路径> 可改用真实语料复测。
const siyuanBenchSeed = `天命之謂性，率性之謂道，修道之謂教。道也者，不可須臾離也，可離非道也。` +
	`是故君子戒慎乎其所不睹，恐懼乎其所不聞。莫見乎隱，莫顯乎微，故君子慎其獨也。` +
	`喜怒哀樂之未發，謂之中；發而皆中節，謂之和。中也者，天下之大本也；和也者，天下之達道也。` +
	`致中和，天地位焉，萬物育焉。學而時習之，不亦說乎？有朋自遠方來，不亦樂乎？` +
	`关关雎鸠，在河之洲。窈窕淑女，君子好逑。参差荇菜，左右流之。求之不得，寤寐思服。` +
	`温故而知新，可以为师矣。学而不思则罔，思而不学则殆。三人行，必有我师焉。` +
	`The quick brown fox jumps over the lazy dog 0123456789 SiYuan FTS benchmark.`

const siyuanBenchChunkRunes = 512

func siyuanBenchChunks(b *testing.B) (chunks []string, bytesPerChunk int64) {
	b.Helper()
	var text string
	if p := os.Getenv("SIYUAN_BENCH_CORPUS"); "" != p {
		data, err := os.ReadFile(p)
		if err != nil {
			b.Fatal(err)
		}
		text = string(data)
	} else {
		var sb strings.Builder
		for sb.Len() < 8<<20 {
			sb.WriteString(siyuanBenchSeed)
		}
		text = sb.String()
	}
	rs := []rune(text)
	var totalBytes int64
	for i := 0; i+siyuanBenchChunkRunes <= len(rs); i += siyuanBenchChunkRunes {
		c := string(rs[i : i+siyuanBenchChunkRunes])
		chunks = append(chunks, c)
		totalBytes += int64(len(c))
	}
	if 16 > len(chunks) {
		b.Fatalf("语料过小: %d chunks", len(chunks))
	}
	return chunks, totalBytes / int64(len(chunks))
}

var siyuanBenchConfigs = []struct {
	name     string
	tokenize string
}{
	{"default", "siyuan"},
	{"case_insensitive", "siyuan case_insensitive"},
	{"han_insensitive", "siyuan han_insensitive"},
	{"case_han_insensitive", "siyuan case_insensitive han_insensitive"},
}

func siyuanBenchOpenFTS(b *testing.B, tokenize string) *sql.DB {
	b.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		b.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	if _, err = db.Exec(fmt.Sprintf(`CREATE VIRTUAL TABLE t USING fts5(content, tokenize="%s")`, tokenize)); err != nil {
		b.Fatal(err)
	}
	return db
}

// BenchmarkSiYuanTokenizerIndexBuild 度量各 tokenize 配置下 FTS5 的索引构建
// 吞吐（含 siyuan 分词与可选折叠、倒排索引更新）。
//
// 每个 op 是一份完全相同的固定工作量：新建 :memory: 表并灌入 8000 个
// 512 字符块（约 400 万字符）——FTS 插入成本随表体量增长，若按"每 op
// 插一条"计量，不同配置的 b.N 不同会引入系统性偏差，固定工作量可避免。
func BenchmarkSiYuanTokenizerIndexBuild(b *testing.B) {
	chunks, bytesPerChunk := siyuanBenchChunks(b)
	const nChunks = 8000
	for len(chunks) < nChunks {
		chunks = append(chunks, chunks...)
	}
	chunks = chunks[:nChunks]
	for _, cfg := range siyuanBenchConfigs {
		b.Run(cfg.name, func(b *testing.B) {
			b.SetBytes(bytesPerChunk * nChunks)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				db := siyuanBenchOpenFTS(b, cfg.tokenize)
				tx, err := db.Begin()
				if err != nil {
					b.Fatal(err)
				}
				stmt, err := tx.Prepare(`INSERT INTO t (content) VALUES (?)`)
				if err != nil {
					b.Fatal(err)
				}
				for _, c := range chunks {
					if _, err = stmt.Exec(c); err != nil {
						b.Fatal(err)
					}
				}
				if err = tx.Commit(); err != nil {
					b.Fatal(err)
				}
				db.Close()
			}
		})
	}
}

// BenchmarkSiYuanTokenizerQuery 度量已建索引上的 MATCH 短语查询延迟。
// trad 查询（不可須臾離）在所有配置下命中；simp 查询（不可须臾离）仅在
// han_insensitive 折叠配置下命中——两者覆盖命中与不命中两种路径。
func BenchmarkSiYuanTokenizerQuery(b *testing.B) {
	chunks, _ := siyuanBenchChunks(b)
	if 20000 < len(chunks) {
		chunks = chunks[:20000]
	}
	queries := []struct{ name, q string }{
		{"trad", `"不可須臾離"`},
		{"simp", `"不可须臾离"`},
	}
	for _, cfg := range siyuanBenchConfigs {
		db := siyuanBenchOpenFTS(b, cfg.tokenize)
		tx, err := db.Begin()
		if err != nil {
			b.Fatal(err)
		}
		stmt, err := tx.Prepare(`INSERT INTO t (content) VALUES (?)`)
		if err != nil {
			b.Fatal(err)
		}
		for _, c := range chunks {
			if _, err = stmt.Exec(c); err != nil {
				b.Fatal(err)
			}
		}
		if err = tx.Commit(); err != nil {
			b.Fatal(err)
		}
		for _, q := range queries {
			b.Run(cfg.name+"/"+q.name, func(b *testing.B) {
				var n int
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					row := db.QueryRow(`SELECT COUNT(*) FROM t WHERE t MATCH ?`, q.q)
					if err := row.Scan(&n); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
		db.Close()
	}
}
