// Copyright 2019 Google LLC
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

package main

import (
	"fmt"
	"open-match.dev/open-match/pkg/pb"
)

// generateProfiles generates test profiles for the matchmaker101 tutorial.
func generateProfiles() []*pb.MatchProfile {
	var profiles []*pb.MatchProfile
	modes := []string{"3vs3"}
	for _, mode := range modes {
		profiles = append(profiles, &pb.MatchProfile{
			Name: "mode_based_profile",
			Pools: []*pb.Pool{
				{
					Name: fmt.Sprintf("mode_%s_all-levels", mode),
					TagPresentFilters: []*pb.TagPresentFilter{
						{
							Tag: mode,
						},
					},
				},
				{
					Name: fmt.Sprintf("mode_%s_level:10-35", mode),
					TagPresentFilters: []*pb.TagPresentFilter{
						{
							Tag: mode,
						},
					},
					DoubleRangeFilters: []*pb.DoubleRangeFilter{
						{
							DoubleArg: "level",
							Min:       10,
							Max:       35,
						},
					},
				},
			},
		},
		)
	}

	return profiles
}
