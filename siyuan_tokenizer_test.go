//go:build sqlite_fts5 || fts5
// +build sqlite_fts5 fts5

package sqlite3

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"
)

// openSiYuanFTS creates an in-memory FTS5 table using the "siyuan"
// tokenizer with the given arguments and inserts the docs.
func openSiYuanFTS(t *testing.T, tokenize string, docs []string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(fmt.Sprintf(
		`CREATE VIRTUAL TABLE t USING fts5(content, tokenize="%s")`, tokenize)); err != nil {
		t.Fatal(err)
	}
	for _, d := range docs {
		if _, err = db.Exec(`INSERT INTO t (content) VALUES (?)`, d); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func ftsMatch(t *testing.T, db *sql.DB, query string) []string {
	t.Helper()
	rows, err := db.Query(`SELECT content FROM t WHERE t MATCH ? ORDER BY rowid`, query)
	if err != nil {
		t.Fatalf("MATCH %q: %v", query, err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var s string
		if err = rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		got = append(got, s)
	}
	return got
}

func expectMatch(t *testing.T, db *sql.DB, query string, want []string) {
	t.Helper()
	got := ftsMatch(t, db, query)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MATCH %q = %#v, want %#v", query, got, want)
	}
}

var siyuanDocs = []string{
	"詩經研究",   // 1 全繁
	"诗经研究",   // 2 全简
	"诗經混排文本", // 3 繁简混排
	"ABC Apple",
	"abc apple",
	"乾隆通寶",
	"出發了",
	"头发",
	"𠀾外字測試", // 9 含 4 字节非 BMP 字符
}

// 默认（无参数）：大小写敏感、繁简敏感 —— 现状行为不变。
func TestSiYuanTokenizerDefault(t *testing.T) {
	db := openSiYuanFTS(t, "siyuan", siyuanDocs)
	defer db.Close()

	expectMatch(t, db, `"诗经"`, []string{"诗经研究"})
	expectMatch(t, db, `"詩經"`, []string{"詩經研究"})
	expectMatch(t, db, `"诗經"`, []string{"诗經混排文本"})
	expectMatch(t, db, `"abc"`, []string{"abc apple"})
	expectMatch(t, db, `"ABC"`, []string{"ABC Apple"})
}

// 仅 case_insensitive：大小写折叠，繁简仍敏感 —— 现状行为不变。
func TestSiYuanTokenizerCaseInsensitive(t *testing.T) {
	db := openSiYuanFTS(t, "siyuan case_insensitive", siyuanDocs)
	defer db.Close()

	expectMatch(t, db, `"abc"`, []string{"ABC Apple", "abc apple"})
	expectMatch(t, db, `"APPLE"`, []string{"ABC Apple", "abc apple"})
	expectMatch(t, db, `"诗经"`, []string{"诗经研究"})
	expectMatch(t, db, `"詩經"`, []string{"詩經研究"})
}

// 仅 han_insensitive：繁简折叠，大小写仍敏感。
func TestSiYuanTokenizerHanInsensitive(t *testing.T) {
	db := openSiYuanFTS(t, "siyuan han_insensitive", siyuanDocs)
	defer db.Close()

	// 简体查询命中繁、简、混排三种文档
	expectMatch(t, db, `"诗经"`, []string{"詩經研究", "诗经研究", "诗經混排文本"})
	// 繁体查询同样命中三种
	expectMatch(t, db, `"詩經"`, []string{"詩經研究", "诗经研究", "诗經混排文本"})
	// 大小写仍然敏感
	expectMatch(t, db, `"abc"`, []string{"abc apple"})
	expectMatch(t, db, `"ABC"`, []string{"ABC Apple"})
	// 單字繁简互通：發/发
	expectMatch(t, db, `"發"`, []string{"出發了", "头发"})
	expectMatch(t, db, `"发"`, []string{"出發了", "头发"})
	// 词组级邻接不破坏：头发 ≠ 出發
	expectMatch(t, db, `"头发"`, []string{"头发"})
	// OpenCC 字级取舍：乾→干，查"干"应命中"乾隆"
	expectMatch(t, db, `"干"`, []string{"乾隆通寶"})
	// 含 4 字节非 BMP 字符的文档可正常索引与命中（測→测折叠）
	expectMatch(t, db, `"测试"`, []string{"𠀾外字測試"})
}

// 两个参数同时启用。
func TestSiYuanTokenizerCaseAndHanInsensitive(t *testing.T) {
	db := openSiYuanFTS(t, "siyuan case_insensitive han_insensitive", siyuanDocs)
	defer db.Close()

	expectMatch(t, db, `"ABC"`, []string{"ABC Apple", "abc apple"})
	expectMatch(t, db, `"诗经"`, []string{"詩經研究", "诗经研究", "诗經混排文本"})
}

// highlight() 的偏移必须落在原文上：折叠只影响匹配，不影响展示。
func TestSiYuanTokenizerHighlightOffsets(t *testing.T) {
	db := openSiYuanFTS(t, "siyuan han_insensitive", siyuanDocs)
	defer db.Close()

	rows, err := db.Query(`SELECT highlight(t, 0, '[', ']') FROM t WHERE t MATCH '"诗经"' ORDER BY rowid`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var s string
		if err = rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		got = append(got, s)
	}
	want := []string{"[詩經]研究", "[诗经]研究", "[诗經]混排文本"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("highlight = %#v, want %#v", got, want)
	}
}

// 空串与纯 ASCII 文本不受影响。
func TestSiYuanTokenizerEdgeCases(t *testing.T) {
	db := openSiYuanFTS(t, "siyuan han_insensitive", []string{"", "plain ascii only", "中"})
	defer db.Close()
	expectMatch(t, db, `"ascii"`, []string{"plain ascii only"})
	expectMatch(t, db, `"中"`, []string{"中"})
}
