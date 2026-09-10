package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

func provisionRule(target string, rule map[string]any) {
	// First, try to delete it to avoid UNIQUE constraint conflicts on restarts
	delReq, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/rules/%s", target, rule["id"]), nil)
	http.DefaultClient.Do(delReq)

	b, _ := json.Marshal(rule)
	req, _ := http.NewRequest("POST", target+"/api/v1/rules", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Failed to provision rule %s: %v\n", rule["id"], err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		fmt.Printf("Server rejected rule %s: HTTP %d\n", rule["id"], resp.StatusCode)
		return
	}
	fmt.Printf("Provisioned Rule: %s\n", rule["name"])
}

func main() {
	target := flag.String("target", "http://localhost:8080", "Engine API URL")
	flag.Parse()

	rand.Seed(time.Now().UnixNano())
	
	// Random Thresholds
	nEntrance := rand.Intn(10) + 5 // 5 to 14
	nCheckout := rand.Intn(5) + 3  // 3 to 7
	nQueueSus := rand.Intn(5) + 3
	nOcc := rand.Intn(10) + 5
	nNoStaff := rand.Intn(5) + 3

	cam := "cam-sim-1"
	
	fmt.Println("======================================")
	fmt.Println("🚀 STARTING ROBUST EDGE SIMULATOR 🚀")
	fmt.Println("======================================")
	fmt.Printf("Threshold N (Entrance Count): %d\n", nEntrance)
	fmt.Printf("Threshold N (Checkout Queue): %d\n", nCheckout)
	fmt.Printf("Threshold N (Queue Sustain):  %d\n", nQueueSus)
	fmt.Printf("Threshold N (Occupancy):      %d\n", nOcc)
	fmt.Printf("Threshold N (No Staff):       %d\n", nNoStaff)
	fmt.Println("======================================")

	// 1. Entrance count ≥ N within 120s
	provisionRule(*target, map[string]any{
		"id": "sim-entrance",
		"name": fmt.Sprintf("SIM: Entrance >= %d in 120s", nEntrance),
		"enabled": true,
		"source": "event",
		"scope": map[string]string{"cameraId": cam, "roiId": "door"},
		"condition": map[string]string{"expression": fmt.Sprintf("window.entranceCount >= %d", nEntrance)},
		"window": map[string]int{"durationSeconds": 120},
		"trigger": map[string]any{"mode": "rising", "cooldownSeconds": 10},
		"action": map[string]string{"type": "sim_alert"},
	})

	// 2. Checkout queue > N
	provisionRule(*target, map[string]any{
		"id": "sim-checkout-inst",
		"name": fmt.Sprintf("SIM: Checkout Queue > %d", nCheckout),
		"enabled": true,
		"source": "state",
		"scope": map[string]string{"cameraId": cam, "roiId": "checkout"},
		"condition": map[string]string{"expression": fmt.Sprintf("occupancy.person > %d", nCheckout)},
		"trigger": map[string]any{"mode": "rising", "cooldownSeconds": 10},
		"action": map[string]string{"type": "sim_alert"},
	})

	// 3. Dwell max > 2 min (120s)
	provisionRule(*target, map[string]any{
		"id": "sim-dwell",
		"name": "SIM: Dwell > 2 mins",
		"enabled": true,
		"source": "state",
		"scope": map[string]string{"cameraId": cam, "roiId": "lobby"},
		"condition": map[string]string{"expression": "dwell.person.max > 120"},
		"trigger": map[string]any{"mode": "rising", "cooldownSeconds": 10},
		"action": map[string]string{"type": "sim_alert"},
	})

	// 4. Queue > N for 3 min (180s)
	provisionRule(*target, map[string]any{
		"id": "sim-queue-sustain",
		"name": fmt.Sprintf("SIM: Queue > %d for 3 mins", nQueueSus),
		"enabled": true,
		"source": "state",
		"scope": map[string]string{"cameraId": cam, "roiId": "queue-zone"},
		"condition": map[string]string{"expression": fmt.Sprintf("occupancy.person > %d", nQueueSus)},
		"sustain": map[string]int{"durationSeconds": 180},
		"trigger": map[string]any{"mode": "rising", "cooldownSeconds": 10},
		"action": map[string]string{"type": "sim_alert"},
	})

	// 5. Occupancy > N
	provisionRule(*target, map[string]any{
		"id": "sim-occupancy",
		"name": fmt.Sprintf("SIM: Occ > %d", nOcc),
		"enabled": true,
		"source": "state",
		"scope": map[string]string{"cameraId": cam, "roiId": "hallway"},
		"condition": map[string]string{"expression": fmt.Sprintf("occupancy.person > %d", nOcc)},
		"trigger": map[string]any{"mode": "rising", "cooldownSeconds": 10},
		"action": map[string]string{"type": "sim_alert"},
	})

	// 6. Occupancy > N AND no staff
	provisionRule(*target, map[string]any{
		"id": "sim-no-staff",
		"name": fmt.Sprintf("SIM: Occ > %d AND No Staff", nNoStaff),
		"enabled": true,
		"source": "state",
		"scope": map[string]string{"cameraId": cam, "roiId": "secure-zone"},
		"condition": map[string]string{"expression": fmt.Sprintf("occupancy.person > %d && staffCount == 0", nNoStaff)},
		"trigger": map[string]any{"mode": "rising", "cooldownSeconds": 10},
		"action": map[string]string{"type": "sim_alert"},
	})

	fmt.Println("\n🌎 Spawning Physics World...")
	world := NewWorld(*target)

	// Start 1Hz State Ticker
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for range ticker.C {
			world.Tick()
		}
	}()

	// SCENARIO ORCHESTRATION
	
	// 1. Entrance Burst Scenario (Prove window works)
	go func() {
		fmt.Printf("▶️  SCENARIO 1: Spawning %d people at the door rapidly...\n", nEntrance + 1)
		for i := 0; i <= nEntrance; i++ {
			world.Spawn(cam, "door")
			time.Sleep(500 * time.Millisecond)
		}
		// Despawn them later
		time.Sleep(10 * time.Second)
		world.Despawn(cam, "door", nEntrance + 1)
	}()

	// 2. Checkout & No Staff Scenarios (Instant state tests)
	go func() {
		time.Sleep(5 * time.Second)
		fmt.Printf("▶️  SCENARIO 2: Flooding checkout (>%d), secure-zone (>%d no staff), hallway (>%d)...\n", nCheckout, nNoStaff, nOcc)
		world.SetStaff(cam, "secure-zone", 1) // initially safe
		for i := 0; i <= nCheckout; i++ { world.Spawn(cam, "checkout") }
		for i := 0; i <= nNoStaff; i++ { world.Spawn(cam, "secure-zone") }
		for i := 0; i <= nOcc; i++ { world.Spawn(cam, "hallway") } // trigger occupancy
		
		time.Sleep(5 * time.Second)
		fmt.Println("▶️  SCENARIO 2b: Staff member leaves secure-zone! Should trigger now.")
		world.SetStaff(cam, "secure-zone", 0) // trigger no-staff rule!
	}()

	// 3. The Sustained Queue & Dwell Scenario (Long running, proves time logic)
	// These require 2 mins (dwell) and 3 mins (queue) to fire!
	go func() {
		time.Sleep(10 * time.Second)
		fmt.Printf("▶️  SCENARIO 3: Spawning 1 guy in lobby (Dwell test) and %d people in queue-zone (3 min sustain test)...\n", nQueueSus + 1)
		fmt.Println("    ⏳ (This will take ~3 minutes to fully resolve...)")
		
		world.Spawn(cam, "lobby")
		for i := 0; i <= nQueueSus; i++ {
			world.Spawn(cam, "queue-zone")
		}
		
		// To prove it must be continuous, let's interrupt the queue after 1 minute, and restart it!
		time.Sleep(60 * time.Second)
		fmt.Println("▶️  SCENARIO 3b: Interruption! Despawning queue-zone. Timer should reset.")
		world.Despawn(cam, "queue-zone", nQueueSus + 1)
		
		time.Sleep(5 * time.Second)
		fmt.Println("▶️  SCENARIO 3c: Respawning queue-zone. 3 minute timer begins NOW.")
		for i := 0; i <= nQueueSus; i++ {
			world.Spawn(cam, "queue-zone")
		}
		
		// Wait 3.5 minutes to guarantee it passes the 120s dwell and 180s queue limits
		time.Sleep(210 * time.Second)
		fmt.Println("✅ All simulations complete.")
	}()

	// Block forever
	select {}
}
