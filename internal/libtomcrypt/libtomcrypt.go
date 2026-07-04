// Copyright (C) 2026 BitBoxSwiss.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//go:build sqlcipher && !libsqlcipher && !darwin
// +build sqlcipher,!libsqlcipher,!darwin

package libtomcrypt

/*
#cgo CFLAGS: -DLTC_SOURCE
#cgo CFLAGS: -DLTC_NOTHING
#cgo CFLAGS: -DLTC_NO_TEST
#cgo CFLAGS: -DLTC_NO_MATH
#cgo CFLAGS: -DLTC_RIJNDAEL
#cgo CFLAGS: -DLTC_CBC_MODE
#cgo CFLAGS: -DLTC_SHA1
#cgo CFLAGS: -DLTC_SHA256
#cgo CFLAGS: -DLTC_SHA512
#cgo CFLAGS: -DLTC_HASH_HELPERS
#cgo CFLAGS: -DLTC_HMAC
#cgo CFLAGS: -DLTC_PKCS_5
#cgo CFLAGS: -DLTC_FORTUNA
#cgo CFLAGS: -DLTC_DEVRANDOM
#cgo CFLAGS: -DLTC_TRY_URANDOM_FIRST
#cgo CFLAGS: -DLTC_RNG_GET_BYTES
#cgo CFLAGS: -I${SRCDIR}
#cgo windows LDFLAGS: -ladvapi32
*/
import "C"
