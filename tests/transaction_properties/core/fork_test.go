// Copyright 2026 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/stretchr/testify/require"
)

func TestOnLatestFork_OnlyTheNewestForkQualifies_WhateverFeaturesItRuns(t *testing.T) {
	var forks []opera.Upgrades
	for _, upgrades := range opera.GetAllHardForksInOrder() {
		forks = append(forks, upgrades)
	}
	require.NotEmpty(t, forks)

	for i, upgrades := range forks[:len(forks)-1] {
		require.False(t, OnLatestFork(upgrades), "fork %d is not the newest", i)
	}

	latest := forks[len(forks)-1]
	require.True(t, OnLatestFork(latest))

	latest.GasSubsidies = true
	latest.TransactionPriorities = true
	latest.TransactionBundles = true
	require.True(t, OnLatestFork(latest), "a feature flag does not change which fork this is")
}
