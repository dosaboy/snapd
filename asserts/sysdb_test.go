// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2015 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package asserts_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	. "gopkg.in/check.v1"

	"github.com/ubuntu-core/snappy/asserts"
	"github.com/ubuntu-core/snappy/dirs"
)

type sysDBSuite struct {
	fakeRoot    string
	probeAssert asserts.Assertion
}

var _ = Suite(&sysDBSuite{})

func (sdbs *sysDBSuite) SetUpTest(c *C) {
	tmpdir := c.MkDir()

	pk := asserts.OpenPGPPrivateKey(testPrivKey0)
	trustedPubKey := pk.PublicKey()
	trustedPubKeyEncoded, err := asserts.EncodePublicKey(trustedPubKey)
	c.Assert(err, IsNil)
	// self-signed
	headers := map[string]string{
		"authority-id":           "canonical",
		"account-id":             "canonical",
		"public-key-id":          trustedPubKey.ID(),
		"public-key-fingerprint": trustedPubKey.Fingerprint(),
		"since":                  "2015-11-20T15:04:00Z",
		"until":                  "2500-11-20T15:04:00Z",
	}
	trustedAccKey, err := asserts.AssembleAndSignInTest(asserts.AccountKeyType, headers, trustedPubKeyEncoded, pk)
	c.Assert(err, IsNil)

	fakeRoot := filepath.Join(tmpdir, "root")
	err = os.Mkdir(fakeRoot, os.ModePerm)
	c.Assert(err, IsNil)

	sdbs.fakeRoot = fakeRoot

	dirs.SetRootDir(sdbs.fakeRoot)
	defer dirs.SetRootDir("/")

	err = os.MkdirAll(filepath.Dir(dirs.SnapTrustedAccountKey), os.ModePerm)
	c.Assert(err, IsNil)
	err = ioutil.WriteFile(dirs.SnapTrustedAccountKey, asserts.Encode(trustedAccKey), os.ModePerm)
	c.Assert(err, IsNil)

	headers = map[string]string{
		"authority-id": "canonical",
		"primary-key":  "0",
	}
	sdbs.probeAssert, err = asserts.AssembleAndSignInTest(asserts.AssertionType("test-only"), headers, nil, pk)
	c.Assert(err, IsNil)
}

func (sdbs *sysDBSuite) TearDownTest(c *C) {
	dirs.SetRootDir("/")
}

func (sdbs *sysDBSuite) TestOpenSysDatabase(c *C) {
	dirs.SetRootDir(sdbs.fakeRoot)
	defer dirs.SetRootDir("/")

	db, err := asserts.OpenSysDatabase()
	c.Assert(err, IsNil)
	c.Check(db, NotNil)

	err = db.Check(sdbs.probeAssert)
	c.Check(err, IsNil)
}

func (sdbs *sysDBSuite) TestOpenSysDatabaseTopCreateFail(c *C) {
	dirs.SetRootDir(sdbs.fakeRoot)
	defer dirs.SetRootDir("/")

	// xxx madness
	// make it not writable
	err := os.RemoveAll(filepath.Dir(filepath.Dir(dirs.SnapAssertsDBDir)))
	c.Assert(err, IsNil)
	err = os.MkdirAll(filepath.Dir(filepath.Dir(dirs.SnapAssertsDBDir)), 0775)
	c.Assert(err, IsNil)
	err = os.MkdirAll(filepath.Dir(dirs.SnapAssertsDBDir), 0555)
	c.Assert(err, IsNil)

	db, err := asserts.OpenSysDatabase()
	c.Assert(err, ErrorMatches, "failed to create assert database root: .*")
	c.Check(db, IsNil)
}

func (sdbs *sysDBSuite) TestOpenSysDatabaseTopCreateFail2(c *C) {
	dirs.SetRootDir(sdbs.fakeRoot)
	defer dirs.SetRootDir("/")

	// xxx madness
	// make it not writable
	err := os.RemoveAll(filepath.Dir(filepath.Dir(dirs.SnapAssertsDBDir)))
	c.Assert(err, IsNil)
	oldUmask := syscall.Umask(0)
	os.MkdirAll(dirs.SnapAssertsDBDir, 0777)
	syscall.Umask(oldUmask)

	db, err := asserts.OpenSysDatabase()
	c.Assert(err, ErrorMatches, "assert storage root unexpectedly world-writable: .*")
	c.Check(db, IsNil)
}
