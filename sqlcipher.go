// Copyright (C) 2022 Jonathan Giannuzzi <jonathan@giannuzzi.me>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//go:build sqlcipher && !libsqlcipher
// +build sqlcipher,!libsqlcipher

package sqlite3

/*
#cgo CFLAGS: -DUSE_SQLCIPHER
#cgo CFLAGS: -DSQLITE_HAS_CODEC
#cgo CFLAGS: -DSQLITE_EXTRA_INIT=sqlcipher_extra_init
#cgo CFLAGS: -DSQLITE_EXTRA_SHUTDOWN=sqlcipher_extra_shutdown
#cgo CFLAGS: -DSQLITE_THREADSAFE=1
#cgo CFLAGS: -DSQLITE_TEMP_STORE=2
*/
import "C"
