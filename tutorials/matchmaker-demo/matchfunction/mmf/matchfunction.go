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

package mmf

import (
	"fmt"
	"log"
	"time"

	"open-match.dev/open-match/pkg/matchfunction"
	"open-match.dev/open-match/pkg/pb"
)

// This match function fetches all the Tickets for all the pools specified in
// the profile. It uses a configured number of tickets from each pool to generate
// a Match Proposal. It continues to generate proposals till one of the pools
// runs out of Tickets.
const (
	matchName              = "basic-matchfunction"
	ticketsPerPoolPerMatch = 6
)

// Run is this match function's implementation of the gRPC call defined in api/matchfunction.proto.
func (s *MatchFunctionService) Run(req *pb.RunRequest, stream pb.MatchFunction_RunServer) error {
	// Fetch tickets for the pools specified in the Match Profile.
	log.Printf("Generating proposals for function %v", req.GetProfile().GetName())

	poolTickets, err := matchfunction.QueryPools(stream.Context(), s.queryServiceClient, req.GetProfile().GetPools())
	if err != nil {
		log.Printf("Failed to query tickets for the given pools, got %s", err.Error())
		return err
	}

	log.Printf("  got %d pools", len(poolTickets))

	// Generate proposals.
	proposals := makeMatchesV2(req.GetProfile(), poolTickets)

	log.Printf("Streaming %v proposals to Open Match", len(proposals))
	// Stream the generated proposals back to Open Match.
	for _, proposal := range proposals {
		if err := stream.Send(&pb.RunResponse{Proposal: proposal}); err != nil {
			log.Printf("Failed to stream proposals to Open Match, got %s", err.Error())
			return err
		}
	}

	return nil
}

func makeMatchesV2(p *pb.MatchProfile, poolTickets map[string][]*pb.Ticket) []*pb.Match {
	var matches []*pb.Match

	for pool, tickets := range poolTickets {
		matches = append(matches, makeMatchesForPool(p, pool, tickets)...)
	}

	return matches
}

func makeMatchesForPool(p *pb.MatchProfile, pool string, tickets []*pb.Ticket) []*pb.Match {
	var matches []*pb.Match
	count := 0

	log.Printf("  in pool %s with %d tickets", pool, len(tickets))

	for len(tickets) >= ticketsPerPoolPerMatch {
		matchTickets := tickets[0:ticketsPerPoolPerMatch]
		tickets = tickets[ticketsPerPoolPerMatch:]

		matches = append(matches, &pb.Match{
			MatchId:       fmt.Sprintf("%v-(%v)-%v#%v", p.GetName(), pool, time.Now().Unix(), count),
			MatchProfile:  p.GetName(),
			MatchFunction: matchName,
			Tickets:       matchTickets,
		})

		count++
	}

	if len(matches) > 0 {
		log.Printf("💃 got %d matches", len(matches))
	}else{
		log.Printf("🤦‍♀️ got %d matches", len(matches))
	}

	return matches
}

func makeMatchesV1(p *pb.MatchProfile, poolTickets map[string][]*pb.Ticket) ([]*pb.Match, error) {
	var matches []*pb.Match
	count := 0

	for {
		insufficientTickets := false
		matchTickets := []*pb.Ticket{}
		for pool, tickets := range poolTickets {
			log.Printf("  in pool %s with %d tickets", pool, len(tickets))

			//for _, ticket := range tickets {
			//	log.Printf("    id:%s level:%f", ticket.Id, ticket.SearchFields.DoubleArgs["level"])
			//}

			if len(tickets) < ticketsPerPoolPerMatch {
				// This pool is completely drained out. Stop creating matches.
				log.Println("👎 INSUFFICIENT TICKETS")
				insufficientTickets = true
				break
			}

			// Remove the Tickets from this pool and add to the match proposal.
			matchTickets = append(matchTickets, tickets[0:ticketsPerPoolPerMatch]...)
			poolTickets[pool] = tickets[ticketsPerPoolPerMatch:]
		}

		if insufficientTickets {
			break
		}

		matches = append(matches, &pb.Match{
			MatchId:       fmt.Sprintf("profile-%v-%v#%v", p.GetName(), time.Now().Unix(), count),
			MatchProfile:  p.GetName(),
			MatchFunction: matchName,
			Tickets:       matchTickets,
		})

		count++
	}

	return matches, nil
}
