// Copyright (c) 2009 The Go Authors. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//    * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//    * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//    * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.

// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package peutil

import (
	"debug/pe"
	"encoding/binary"
)

// Export describes a single export entry.
type Export struct {
	Name string
}

// Exports returns the names of known named exports.
func (f *File) Exports() ([]Export, error) {
	_, pe64 := f.OptionalHeader.(*pe.OptionalHeader64)

	// grab the number of data directory entries
	var ddLength uint32
	if pe64 {
		ddLength = f.OptionalHeader.(*pe.OptionalHeader64).NumberOfRvaAndSizes
	} else {
		ddLength = f.OptionalHeader.(*pe.OptionalHeader32).NumberOfRvaAndSizes
	}

	// check that the length of data directory entries is large
	// enough to include the exports directory.
	if ddLength < pe.IMAGE_DIRECTORY_ENTRY_EXPORT+1 {
		return nil, nil
	}

	// grab the export data directory entry
	var edd pe.DataDirectory
	if pe64 {
		edd = f.OptionalHeader.(*pe.OptionalHeader64).DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXPORT]
	} else {
		edd = f.OptionalHeader.(*pe.OptionalHeader32).DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXPORT]
	}

	// figure out which section contains the export directory table
	var ds *pe.Section
	ds = nil
	for _, s := range f.Sections {
		if s.VirtualAddress <= edd.VirtualAddress && edd.VirtualAddress < s.VirtualAddress+s.VirtualSize {
			ds = s
			break
		}
	}

	// didn't find a section, so no exports were found
	if ds == nil {
		return nil, nil
	}

	d, err := ds.Data()
	if err != nil {
		return nil, err
	}

	offset := edd.VirtualAddress - ds.VirtualAddress

	// seek to the virtual address specified in the export data directory
	dxd := d[offset:]

	// seek to names table
	dnn := d[binary.LittleEndian.Uint32(dxd[32:36])-ds.VirtualAddress:]

	names := binary.LittleEndian.Uint32(dxd[24:28])
	exports := make([]Export, names)
	for i := range names {
		nameRVA := binary.LittleEndian.Uint32(dnn[i*4 : (i*4)+4])

		name, _ := getString(d, int(nameRVA-ds.VirtualAddress))
		exports = append(exports, Export{
			Name: name,
		})
	}

	return exports, nil
}

// getString extracts a string from symbol string table.
func getString(section []byte, start int) (string, bool) {
	if start < 0 || start >= len(section) {
		return "", false
	}

	for end := start; end < len(section); end++ {
		if section[end] == 0 {
			return string(section[start:end]), true
		}
	}
	return "", false
}
