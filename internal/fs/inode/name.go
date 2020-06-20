// Copyright 2020 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package inode

import (
	"fmt"
)

// Name is the inode's name that can be interpreted in 2 ways:
//   (1) LocalName: the name of the inode in the local file system.
//   (2) GcsObjectName: the name of its gcs object backed by the inode.
type Name struct {
	// The gcs object's name in its bucket.
	objectName string
}

// NewRootName creates a Name for the root directory of a gcs bucket
func NewRootName() Name {
	return Name{""}
}

// NewDirName creates a new inode name for a directory.
func NewDirName(parentName Name, dirName string) Name {
	if parentName.IsFile() || dirName == "" {
		panic(fmt.Sprintf(
			"Inode '%s' cannot have child subdirectory '%s'",
			parentName,
			dirName))
	}
	if dirName[len(dirName)-1] != '/' {
		dirName = dirName + "/"
	}
	return Name{parentName.objectName + dirName}
}

// NewFileName creates a new inode name for a file.
func NewFileName(parentName Name, fileName string) Name {
	if parentName.IsFile() || fileName == "" {
		panic(fmt.Sprintf(
			"Inode '%s' cannot have child file '%s'",
			parentName,
			fileName))
	}
	return Name{parentName.objectName + fileName}
}

// IsDir returns true if the name represents a directory.
func (name Name) IsDir() bool {
	isBucketRoot := (name.objectName == "")
	return isBucketRoot || name.objectName[len(name.objectName)-1] == '/'
}

// IsFile returns true if the name represents a file.
func (name Name) IsFile() bool {
	return !name.IsDir()
}

// GcsObjectName returns the name of the gcs object backed by the inode.
func (name Name) GcsObjectName() string {
	return name.objectName
}

// LocalName returns the name of the directory or file in the mounted file system.
func (name Name) LocalName() string {
	return name.objectName
}

// String returns LocalName.
func (name Name) String() string {
	return name.LocalName()
}
