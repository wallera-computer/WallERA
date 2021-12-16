// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package nonsecuresyscall

const (
	SYS_EXIT = iota
	SYS_WRITE
	SYS_NANOTIME
	SYS_GETRANDOM
	SYS_RPC_REQ
	SYS_RPC_RES
)
