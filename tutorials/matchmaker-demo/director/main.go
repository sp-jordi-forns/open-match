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
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
	"open-match.dev/open-match/pkg/pb"
)

// The Director in this tutorial continously polls Open Match for the Match
// Profiles and makes random assignments for the Tickets in the returned matches.

const (
	// The endpoint for the Open Match Backend service.
	omBackendEndpoint = "om-backend.open-match.svc.cluster.local:50505"
	// The Host and Port for the Match Function service endpoint.
	functionHostName       = "mm-demo-matchfunction.mm-demo.svc.cluster.local"
	functionPort     int32 = 50502

	tickDuration = 5 * time.Second
)

func main() {
	log.Println("☎️ Dialing...")

	// Connect to Open Match Backend.
	conn, err := grpc.Dial(omBackendEndpoint, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to Open Match Backend, got %s", err.Error())
	}

	defer conn.Close()
	be := pb.NewBackendServiceClient(conn)

	log.Println("💤 Sleeping...")

	time.Sleep(10 * time.Second)

	run(be)

	for range time.Tick(tickDuration) {
		run(be)
	}
}

func run(be pb.BackendServiceClient) {
	profiles := generateProfiles()

	log.Printf("🔎️ Fetching matches for %v profiles", len(profiles))

	// Fetch matches for each profile and make random assignments for Tickets in
	// the matches returned.
	var wg sync.WaitGroup
	for _, p := range profiles {
		wg.Add(1)
		go func(wg *sync.WaitGroup, p *pb.MatchProfile) {
			defer wg.Done()
			matches, err := fetch(be, p)
			if err != nil {
				log.Printf("Failed to fetch matches for profile %v, got %s", p.GetName(), err.Error())
				return
			}

			if len(matches) == 0 {
				log.Printf("🤷‍♀️️ Generated %v matches for profile %v", len(matches), p.GetName())
			} else {
				log.Printf("✌️ Generated %v matches for profile %v", len(matches), p.GetName())
			}

			if err := assign(be, matches); err != nil {
				log.Printf("Failed to assign servers to matches, got %s", err.Error())
				return
			}
		}(&wg, p)
	}

	wg.Wait()
}

func fetch(be pb.BackendServiceClient, p *pb.MatchProfile) ([]*pb.Match, error) {
	req := &pb.FetchMatchesRequest{
		Config: &pb.FunctionConfig{
			Host: functionHostName,
			Port: functionPort,
			Type: pb.FunctionConfig_GRPC,
		},
		Profile: p,
	}

	stream, err := be.FetchMatches(context.Background(), req)
	if err != nil {
		log.Println()
		return nil, err
	}

	var result []*pb.Match
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		result = append(result, resp.GetMatch())
	}

	return result, nil
}

func assign(be pb.BackendServiceClient, matches []*pb.Match) error {
	for _, match := range matches {
		ticketIDs := []string{}
		for _, t := range match.GetTickets() {
			ticketIDs = append(ticketIDs, t.Id)
		}

		conn := match.MatchId
		req := &pb.AssignTicketsRequest{
			TicketIds: ticketIDs,
			Assignment: &pb.Assignment{
				Connection: conn,
			},
		}

		if _, err := be.AssignTickets(context.Background(), req); err != nil {
			return fmt.Errorf("AssignTickets failed for match %v, got %w", match.GetMatchId(), err)
		}

		log.Printf("👾 Assigned server %v to match %v, containing %d tickets", conn, match.GetMatchId(), len(match.Tickets))
	}

	return nil
}
