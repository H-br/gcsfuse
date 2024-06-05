// Copyright 2024 Google Inc. All Rights Reserved.
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

// A collection of tests for a file system where parallel dirops are allowed.
// Dirops refers to readdir and lookup operations.

package fs_test

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	kernelListCacheTtlSeconds = 1000
)

// Base of the all tests:
// How do we detect if the ReadDir() is responded from gcsfuse-filesystem or kernel?
// By ensuring different response of same ReadDir() call from gcsfuse-filesystem.
// So, if the current ReadDir() response is matching with previous one then
// we can clearly say it is served from kernel cache. If not matching then served
// from gcsfuse filesystem.
type KernelListCacheTestCommon struct {
	suite.Suite
	fsTest
}

func (t *KernelListCacheTestCommon) SetupSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.MountConfig = &config.MountConfig{
		FileSystemConfig: config.FileSystemConfig{
			DisableParallelDirops:     false,
			KernelListCacheTtlSeconds: kernelListCacheTtlSeconds,
		}}
	t.serverCfg.RenameDirLimit = 10
	t.fsTest.SetUpTestSuite()
}

func (t *KernelListCacheTestCommon) SetupTest() {
	t.createFilesAndDirStructureInBucket()
	cacheClock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))
}

func (t *KernelListCacheTestCommon) TearDownTest() {
	cacheClock.AdvanceTime(util.MaxTimeDuration)
	t.fsTest.TearDown()
	//os.RemoveAll(path.Join(mntDir))
}

func (t *KernelListCacheTestCommon) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func TestKernelListCacheTestSuite(t *testing.T) {
	suite.Run(t, new(KernelListCacheTestCommon))
}

// createFilesAndDirStructureInBucket creates the following files and directory
// structure.
// bucket
//
//	explicitDir/
//	explicitDir/file1.txt
//	explicitDir/file2.txt
//	implicitDir/file1.txt
//	implicitDir/file2.txt
func (t *KernelListCacheTestCommon) createFilesAndDirStructureInBucket() {
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/":          "",
		"explicitDir/file1.txt": "12345",
		"explicitDir/file2.txt": "6789101112",
		"implicitDir/file1.txt": "-1234556789",
		"implicitDir/file2.txt": "kdfkdj9",
	}))
}

type KernelListCacheTestWithPositiveTtl struct {
	suite.Suite
	fsTest
}

func (t *KernelListCacheTestWithPositiveTtl) SetupSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.MountConfig = &config.MountConfig{
		FileSystemConfig: config.FileSystemConfig{
			DisableParallelDirops:     false,
			KernelListCacheTtlSeconds: kernelListCacheTtlSeconds,
		}}
	t.serverCfg.RenameDirLimit = 10
	t.fsTest.SetUpTestSuite()
}

func (t *KernelListCacheTestWithPositiveTtl) SetupTest() {
	t.createFilesAndDirStructureInBucket()
	cacheClock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))
}

func (t *KernelListCacheTestWithPositiveTtl) TearDownTest() {
	cacheClock.AdvanceTime(util.MaxTimeDuration)
	t.fsTest.TearDown()
	//os.RemoveAll(path.Join(mntDir))
}

func (t *KernelListCacheTestWithPositiveTtl) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func TestKernelListCacheTestSuite(t *testing.T) {
	suite.Run(t, new(KernelListCacheTestWithPositiveTtl))
}

// createFilesAndDirStructureInBucket creates the following files and directory
// structure.
// bucket
//
//	explicitDir/
//	explicitDir/file1.txt
//	explicitDir/file2.txt
//	implicitDir/file1.txt
//	implicitDir/file2.txt
func (t *KernelListCacheTestWithPositiveTtl) createFilesAndDirStructureInBucket() {
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/":          "",
		"explicitDir/file1.txt": "12345",
		"explicitDir/file2.txt": "6789101112",
		"implicitDir/file1.txt": "-1234556789",
		"implicitDir/file2.txt": "kdfkdj9",
	}))
}

// TestKernelListCache_SimpleCacheWorking:
// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() within ttl will be served from kernel page cache.
func (t *KernelListCacheTestWithPositiveTtl) TestKernelListCache_CacheHit() {
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	defer f.Close()

	names1, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 2, len(names1))
	assert.Equal(t.T(), names1[0], "file1.txt")
	assert.Equal(t.T(), names1[1], "file2.txt")

	err = f.Close()
	assert.Nil(t.T(), err)

	// Adding one object to make sure to change the ReadDir() response.
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/file3.txt": "123456",
	}))
	defer t.deleteObject("explicitDir/file3.txt")

	// Advancing the clock within time.
	cacheClock.AdvanceTime(kernelListCacheTtlSeconds * time.Second / 2)

	// First read, kernel will cache the dir response.
	f, err = os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	names2, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 2, len(names2))
	assert.Equal(t.T(), names1[0], names2[0])
	assert.Equal(t.T(), names1[1], names2[1])

	err = f.Close()
	assert.Nil(t.T(), err)
}

// TestKernelListCache_CacheMissOutOfTtl
// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() out of ttl will also be served from GCSFuse filesystem.
func (t *KernelListCacheTestWithPositiveTtl) TestKernelListCache_CacheMiss() {
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	defer f.Close()

	names1, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 2, len(names1))
	assert.Equal(t.T(), names1[0], "file1.txt")
	assert.Equal(t.T(), names1[1], "file2.txt")

	err = f.Close()
	assert.Nil(t.T(), err)

	// Adding one object to make sure to change the ReadDir() response.
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/file3.txt": "123456",
	}))
	defer t.deleteObject("explicitDir/file3.txt")

	// Advancing the time more than ttl.
	cacheClock.AdvanceTime(kernelListCacheTtlSeconds*time.Second + time.Second)

	// Since out of ttl, so invalidation happens and ReadDir() will be served from
	// gcsfuse filesystem.
	f, err = os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	names2, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 3, len(names2))
	assert.Equal(t.T(), names2[0], "file1.txt")
	assert.Equal(t.T(), names2[1], "file2.txt")
	assert.Equal(t.T(), names2[2], "file3.txt")

	err = f.Close()
	assert.Nil(t.T(), err)
}

// TestKernelListCache_CacheHitAfterInvalidation:
// (a) First read will be served from GcsFuse filesystem.
// (b) Second read after ttl will also be served from GCSFuse file-system.
// (c) Third read within ttl will be served from kernel page cache.
func (t *KernelListCacheTestWithPositiveTtl) TestKernelListCache_CacheHitAfterInvalidation() {
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	defer f.Close()

	names1, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 2, len(names1))
	assert.Equal(t.T(), names1[0], "file1.txt")
	assert.Equal(t.T(), names1[1], "file2.txt")

	err = f.Close()
	assert.Nil(t.T(), err)

	// Adding one object to make sure to change the ReadDir() response.
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/file3.txt": "123456",
	}))
	defer t.deleteObject("explicitDir/file3.txt")

	// Advancing the time more than ttl.
	cacheClock.AdvanceTime(kernelListCacheTtlSeconds*time.Second + time.Second)

	// Since out of ttl, so invalidation happens and ReadDir() will be served from
	// gcsfuse filesystem.
	f, err = os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	names2, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 3, len(names2))
	assert.Equal(t.T(), names2[0], "file1.txt")
	assert.Equal(t.T(), names2[1], "file2.txt")
	assert.Equal(t.T(), names2[2], "file3.txt")

	err = f.Close()
	assert.Nil(t.T(), err)

	// Adding one object to make sure to change the ReadDir() response.
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/file4.txt": "123456",
	}))
	defer t.deleteObject("explicitDir/file4.txt")

	// Advancing the time within ttl.
	cacheClock.AdvanceTime(kernelListCacheTtlSeconds * time.Second / 2)

	// Within ttl, so will be served from kernel.
	f, err = os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	names3, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 3, len(names3))
	assert.Equal(t.T(), names2[0], names3[0])
	assert.Equal(t.T(), names2[1], names3[1])
	assert.Equal(t.T(), names2[2], names3[2])

	err = f.Close()
	assert.Nil(t.T(), err)
}

type KernelListCacheTestWithInfiniteTtl struct {
	suite.Suite
	fsTest
}

func (t *KernelListCacheTestWithInfiniteTtl) SetupSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.MountConfig = &config.MountConfig{
		FileSystemConfig: config.FileSystemConfig{
			DisableParallelDirops:     false,
			KernelListCacheTtlSeconds: -1,
		}}
	t.serverCfg.RenameDirLimit = 10
	t.fsTest.SetUpTestSuite()
}

func (t *KernelListCacheTestWithInfiniteTtl) SetupTest() {
	t.createFilesAndDirStructureInBucket()
}

func (t *KernelListCacheTestWithInfiniteTtl) TearDownTest() {
	t.fsTest.TearDown()
}

func (t *KernelListCacheTestWithInfiniteTtl) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func TestKernelListCacheTestInfiniteTtlSuite(t *testing.T) {
	suite.Run(t, new(KernelListCacheTestWithInfiniteTtl))
}

// TestKernelListCache_CacheMissWithZeroTtl
// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() will also be served from GCSFuse filesystem.
func (t *KernelListCacheTestWithInfiniteTtl) TestKernelListCache_CacheHit() {
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	defer f.Close()

	names1, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 2, len(names1))
	assert.Equal(t.T(), names1[0], "file1.txt")
	assert.Equal(t.T(), names1[1], "file2.txt")

	err = f.Close()
	assert.Nil(t.T(), err)

	// Adding one object to make sure to change the ReadDir() response.
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/file3.txt": "123456",
	}))
	defer t.deleteObject("explicitDir/file3.txt")

	// Advancing time by 5 years (157800000 seconds).
	cacheClock.AdvanceTime(157800000 * time.Second)

	// No invalidation since infinite ttl.
	f, err = os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	names2, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 2, len(names2))
	assert.Equal(t.T(), names1[0], names2[0])
	assert.Equal(t.T(), names1[1], names2[1])

	err = f.Close()
	assert.Nil(t.T(), err)
}

// createFilesAndDirStructureInBucket creates the following files and directory
// structure.
// bucket
//
//	explicitDir/
//	explicitDir/file1.txt
//	explicitDir/file2.txt
//	implicitDir/file1.txt
//	implicitDir/file2.txt
func (t *KernelListCacheTestWithInfiniteTtl) createFilesAndDirStructureInBucket() {
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/":          "",
		"explicitDir/file1.txt": "12345",
		"explicitDir/file2.txt": "6789101112",
		"implicitDir/file1.txt": "-1234556789",
		"implicitDir/file2.txt": "kdfkdj9",
	}))
}

type KernelListCacheTestWithZeroTtl struct {
	suite.Suite
	fsTest
}

func (t *KernelListCacheTestWithZeroTtl) SetupSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.MountConfig = &config.MountConfig{
		FileSystemConfig: config.FileSystemConfig{
			DisableParallelDirops:     false,
			KernelListCacheTtlSeconds: 0,
		}}
	t.serverCfg.RenameDirLimit = 10
	t.fsTest.SetUpTestSuite()
}

func (t *KernelListCacheTestWithZeroTtl) SetupTest() {
	t.createFilesAndDirStructureInBucket()
}

func (t *KernelListCacheTestWithZeroTtl) TearDownTest() {
	t.fsTest.TearDown()
}

func (t *KernelListCacheTestWithZeroTtl) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func TestKernelListCacheTestZeroTtlSuite(t *testing.T) {
	suite.Run(t, new(KernelListCacheTestWithZeroTtl))
}

// createFilesAndDirStructureInBucket creates the following files and directory
// structure.
// bucket
//
//	explicitDir/
//	explicitDir/file1.txt
//	explicitDir/file2.txt
//	implicitDir/file1.txt
//	implicitDir/file2.txt
func (t *KernelListCacheTestWithZeroTtl) createFilesAndDirStructureInBucket() {
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/":          "",
		"explicitDir/file1.txt": "12345",
		"explicitDir/file2.txt": "6789101112",
		"implicitDir/file1.txt": "-1234556789",
		"implicitDir/file2.txt": "kdfkdj9",
	}))
}

// TestKernelListCache_CacheMissWithZeroTtl
// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() will also be served from GCSFuse filesystem.
func (t *KernelListCacheTestWithZeroTtl) TestKernelListCache_CacheMiss() {
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	defer f.Close()

	names1, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 2, len(names1))
	assert.Equal(t.T(), names1[0], "file1.txt")
	assert.Equal(t.T(), names1[1], "file2.txt")

	err = f.Close()
	assert.Nil(t.T(), err)

	// Adding one object to make sure to change the ReadDir() response.
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/file3.txt": "123456",
	}))
	defer t.deleteObject("explicitDir/file3.txt")

	// Zero ttl, means readdir will always be served from gcsfuse.
	f, err = os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	names2, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 3, len(names2))
	assert.Equal(t.T(), names2[0], "file1.txt")
	assert.Equal(t.T(), names2[1], "file2.txt")
	assert.Equal(t.T(), names2[2], "file3.txt")

	err = f.Close()
	assert.Nil(t.T(), err)
}
