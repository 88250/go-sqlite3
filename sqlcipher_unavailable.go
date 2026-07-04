// Copyright (C) 2026 BitBoxSwiss
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//go:build !sqlcipher && !libsqlcipher
// +build !sqlcipher,!libsqlcipher

package sqlite3

const sqlcipherAvailable = false
