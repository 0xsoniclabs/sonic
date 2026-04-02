package emitter

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensusengine"
	"github.com/0xsoniclabs/consensus/consensus/consensustest"
	"github.com/0xsoniclabs/sonic/emitter/ancestor"
)

type Results struct {
	maxFrame  consensus.Frame
	numEvents int
}

type latency interface {
	latency(sender int, receiver int, rng *rand.Rand) int
}

type gaussianLatency struct {
	mean float64
	std  float64 // standard deviation
}

type QITestEvents []*QITestEvent

type QITestEvent struct {
	consensustest.TestEvent
	creationTime int
}

var mutex sync.Mutex // a mutex used for variables shared across go rountines

func Benchmark_Emission(b *testing.B) {
	numNodes := 20
	stakeDist := stakeCumDist()             // for stakes drawn from distribution
	stakeRNG := rand.New(rand.NewSource(0)) // for stakes drawn from distribution

	weights := make([]consensus.Weight, numNodes)
	for i := range weights {
		// uncomment one of the below options for valiator stake distribution
		weights[i] = consensus.Weight(1)                               //for equal stake
		weights[i] = consensus.Weight(sampleDist(stakeRNG, stakeDist)) // for non-equal stake sample from Sonic main net validator stake distribution
	}
	sort.Slice(weights, func(i, j int) bool { return weights[i] > weights[j] }) // sort weights in order
	QIParentCount := 12                                                         // maximum number of parents selected by FC indexer
	randParentCount := 0                                                        // maximum number of parents selected randomly
	offlineNodes := false                                                       // set to true to make smallest non-quourm validators offline

	// Uncomment the desired latency type

	// Latencies between validators are drawn from a Normal Gaussian distribution
	var latency gaussianLatency
	latency.mean = 100 // mean latency in milliseconds
	latency.std = 10   // standard deviation of latency in milliseconds
	maxLatency := int(latency.mean + 4*latency.std)

	// Latencies between validators are modelled using a dataset of real world internet latencies between cities
	// var latency cityLatency
	// var seed int64
	// seed = 0 //use this for the same seed each time the simulator runs
	// // seed = time.Now().UnixNano() //use this for a different seed each time the simulator runs
	// maxLatency := latency.initialise(numNodes, seed)

	// Latencies between validators are drawn from a dataset of latencies observed by one Sonic main net validator. Note all pairs of validators will use the same distribution
	// var latency mainNetLatency
	// maxLatency := latency.initialise()

	simulationDuration := 50000 // length of simulated time in milliseconds
	thresholds := make([]float64, 0)
	for thresh := 0.0; thresh <= 1000.0; thresh = thresh + 50.0 {
		thresholds = append(thresholds, thresh)
	}
	results := make([]Results, len(thresholds))
	for ti, threshold := range thresholds {
		fmt.Println("Threshold: ", threshold)
		//Now run the simulation
		results[ti] = simulate(weights, QIParentCount, randParentCount, offlineNodes, &latency, maxLatency, simulationDuration)
	}

	// Print Results
	currenTime := time.Now()
	fileName := "../../../../SimulationResults "
	fileName += currenTime.String()
	fileName += ".txt"
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	mw := io.MultiWriter(os.Stdout, file)

	_, _ = fmt.Fprint(mw, "maxFrame=np.array([")
	for _, result := range results {
		_, _ = fmt.Fprint(mw, result.maxFrame)
	}
	_, _ = fmt.Fprintln(mw, "])")
	_, _ = fmt.Fprintln(mw, "")

	_, _ = fmt.Fprint(mw, "numEvents=np.array([")
	for _, result := range results {
		_, _ = fmt.Fprint(mw, result.numEvents)
	}
	_, _ = fmt.Fprintln(mw, "])")
	_, _ = fmt.Fprintln(mw, "")

	_, _ = fmt.Fprint(mw, "thresholds=np.array([")
	for ti := range results {
		_, _ = fmt.Fprint(mw, thresholds[ti])
	}
	_, _ = fmt.Fprintln(mw, "])")
	_, _ = fmt.Fprintln(mw, "")
	_, _ = fmt.Fprintln(mw, "")

}

func simulate(weights []consensus.Weight, QIParentCount int, randParentCount int, offlineNodes bool, latency latency, maxLatency int, simulationDuration int) Results {

	numValidators := len(weights)

	randSrc := rand.New(rand.NewSource(0)) // use a fixed seed of 0 for comparison between runs

	latencyRNG := make([]*rand.Rand, numValidators)
	randParentRNG := make([]*rand.Rand, numValidators)
	randEvRNG := make([]*rand.Rand, numValidators)
	for i := range weights {
		// Use same seed each time the simulator is used
		latencyRNG[i] = rand.New(rand.NewSource(0))
		randParentRNG[i] = rand.New(rand.NewSource(0))
		randEvRNG[i] = rand.New(rand.NewSource(0))

		// Uncomment to use a different seed each time the simulator is used
		// time.Sleep(1 * time.Millisecond) //sleep a bit for seeding RNG
		// latencyRNG[i] = rand.New(rand.NewSource(time.Now().UnixNano()))
		// time.Sleep(1 * time.Millisecond) //sleep a bit for seeding RNG
		// randParentRNG[i] = rand.New(rand.NewSource(time.Now().UnixNano()))
		// time.Sleep(1 * time.Millisecond) //sleep a bit for seeding RNG
		// randEvRNG[i] = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	randEvRate := 0.0 // sets the probability that an event will be created randomly

	// create a 3D slice with coordinates [time][node][node] that is used to store delayed transmission of events between nodes
	//each time coordinate corresponds to 1 millisecond of delay between a pair of nodes
	eventPropagation := make([][][][]*QITestEvent, maxLatency)
	for i := range eventPropagation {
		eventPropagation[i] = make([][][]*QITestEvent, numValidators)
		for j := range eventPropagation[i] {
			eventPropagation[i][j] = make([][]*QITestEvent, numValidators)
			for k := range eventPropagation[i][j] {
				eventPropagation[i][j][k] = make([]*QITestEvent, 0)
			}
		}
	}

	// create a list of heads for each node
	headsAll := make([]consensus.Events, numValidators)

	//setup nodes
	nodes := consensustest.GenNodes(numValidators)
	validators := consensus.ArrayToValidators(nodes, weights)

	var input *consensustest.TestEventSource
	var lch *consensusengine.CoreLachesis
	var dagIndexer ancestor.DagIndex
	inputs := make([]consensustest.TestEventSource, numValidators)
	lchs := make([]consensusengine.CoreLachesis, numValidators)
	fcIndexers := make([]*ancestor.FCIndexer, numValidators)
	for i := 0; i < numValidators; i++ {
		lch, _, input, dagIndexer = consensusengine.NewBootstrappedCoreConsensus(nodes, weights)
		lchs[i] = *lch
		inputs[i] = *input
		fcIndexers[i] = ancestor.NewFCIndexer(validators, dagIndexer, nodes[i])
	}

	// If required set smallest non-quorum validators as offline for testing
	sortWeights := validators.SortedWeights()
	sortedIDs := validators.SortedIDs()
	onlineStake := validators.TotalWeight()
	online := make(map[consensus.ValidatorID]bool)
	for i := len(sortWeights) - 1; i >= 0; i-- {
		online[sortedIDs[i]] = true
		if offlineNodes {
			if onlineStake-sortWeights[i] >= validators.Quorum() {
				onlineStake -= sortWeights[i]
				online[sortedIDs[i]] = false
			}
		}
	}
	minCheckInterval := 11 // min interval before re-checking if event can be created
	prevCheckTime := make([]int, numValidators)
	minEventCreationInterval := make([]int, numValidators) // minimum interval between creating event
	for i := range minEventCreationInterval {
		// minEventCreationInterval[i] = int(30 * float64(weights[0]) / float64(weights[i]))
		minEventCreationInterval[i] = 11
	}
	// initial delay to avoid synchronous events
	initialDelay := make([]int, numValidators)
	for i := range initialDelay {
		initialDelay[i] = randSrc.Intn(maxLatency)
	}

	bufferedEvents := make([]QITestEvents, numValidators)

	eventsComplete := make([]int, numValidators)

	// setup flag to indicate leaf event
	isLeaf := make([]bool, numValidators)
	for node := range isLeaf {
		isLeaf[node] = true
	}

	selfParent := make([]QITestEvent, numValidators)

	wg := sync.WaitGroup{} // used for parallel go routines

	timeIdx := maxLatency - 1 // circular buffer time index
	simTime := -1             // counts simulated time

	// now start the simulation
	for simTime < simulationDuration {
		// move forward one timestep
		timeIdx = (timeIdx + 1) % maxLatency
		simTime = simTime + 1
		if simTime%1000 == 0 {
			fmt.Print(" TIME: ", simTime) // print time progress for tracking simulation progression
		}

		// Check to see if new events are received by nodes
		// if they are, do the appropriate updates for the received event
		for receiveNode := 0; receiveNode < numValidators; receiveNode++ {
			wg.Add(1)
			go func(receiveNode int) {
				defer wg.Done()
				// check for events to be received by other nodes (including self)
				for sendNode := 0; sendNode < numValidators; sendNode++ {
					mutex.Lock()
					for i := 0; i < len(eventPropagation[timeIdx][sendNode][receiveNode]); i++ {
						e := eventPropagation[timeIdx][sendNode][receiveNode][i]
						//add new event to buffer for cheecking if events are ready to put in DAG
						bufferedEvents[receiveNode] = append(bufferedEvents[receiveNode], e)
					}
					//clear the events at this time index

					eventPropagation[timeIdx][sendNode][receiveNode] = eventPropagation[timeIdx][sendNode][receiveNode][:0]
					mutex.Unlock()
				}
				// it is required that all of an event's parents have been received before adding to DAG
				// loop through buffer to check for events that can be processed
				process := make([]bool, len(bufferedEvents[receiveNode]))
				for i, buffEvent := range bufferedEvents[receiveNode] {
					process[i] = true
					//check if all parents are in the DAG
					for _, parent := range buffEvent.Parents() {
						if lchs[receiveNode].Input.GetEvent(parent) == nil {
							// a parent is not yet in the DAG, so don't process this event yet
							process[i] = false
							break
						}
					}
					if process[i] {
						// buffered event has all parents in the DAG and can now be processed
						processEvent(inputs[receiveNode], &lchs[receiveNode], buffEvent, fcIndexers[receiveNode], &headsAll[receiveNode], nodes[receiveNode], simTime)
					}
				}
				//remove processed events from buffer
				temp := make([]*QITestEvent, len(bufferedEvents[receiveNode]))
				copy(temp, bufferedEvents[receiveNode])
				bufferedEvents[receiveNode] = bufferedEvents[receiveNode][:0] //clear buffer
				for i, processed := range process {
					if processed == false {
						bufferedEvents[receiveNode] = append(bufferedEvents[receiveNode], temp[i]) // put unprocessed event back in the buffer
					}

				}
			}(receiveNode)
		}
		wg.Wait()

		// Build events and check timing condition
		for self := 0; self < numValidators; self++ {
			passedTime := simTime - prevCheckTime[self] // time since creating previous event
			if passedTime >= minCheckInterval {
				prevCheckTime[self] = simTime
				// self is ready to try creating a new event
				wg.Add(1)
				go func(self int) { //parallel
					defer wg.Done()

					if initialDelay[self] > 0 {
						// don't create an event during an initial delay in creating the first event at the start of the simulation
						initialDelay[self]--
					} else {
						//create the event datastructure
						selfID := nodes[self]
						e := &QITestEvent{}
						e.SetCreator(selfID)
						e.SetParents(consensus.EventHashes{}) // first parent is empty hash

						var parents consensus.Events
						if isLeaf[self] { // leaf event
							e.SetSeq(1)
							e.SetLamport(1)
						} else { // normal event
							e.SetSeq(selfParent[self].Seq() + 1)
							e.SetLamport(selfParent[self].Lamport() + 1)
							parents = append(parents, &selfParent[self].BaseEvent) // always use self's previous event as a parent
						}

						// get heads for parent selection
						heads := append(consensus.Events{}, headsAll[self]...)
						for i, head := range heads {
							if selfParent[self].ID() == head.ID() {
								// remove the self parent from options, it is already a parent
								heads[i] = heads[len(heads)-1]
								heads = heads[:len(heads)-1]
								break
							}
						}

						//fcIndexers[self].SelfParentEvent = selfParent[self].ID() // fcIndexer needs to know the self's previous event
						if !isLeaf[self] { // only non leaf events have parents
							// iteratively select the best parent from the list of heads using quorum indexer parent selection
							for j := 0; j < QIParentCount-1; j++ {
								if len(heads) <= 0 {
									//no more heads to choose, adding more parents will not improve DAG progress
									break
								}

								best := fcIndexers[self].SearchStrategy().Choose(parents.IDs(), heads.IDs()) //new fcIndexer

								parents = append(parents, heads[best])
								// remove chosen parent from head options
								heads[best] = heads[len(heads)-1]
								heads = heads[:len(heads)-1]
							}

							// now select random parents
							for j := 0; j < randParentCount-1; j++ {
								if len(heads) <= 0 {
									//no more heads to choose, adding more parents will not improve DAG progress
									break
								}
								randParent := randParentRNG[self].Intn(len(heads))
								parents = append(parents, heads[randParent])
								// remove chosen parent from head options
								heads[randParent] = heads[len(heads)-1]
								heads = heads[:len(heads)-1]
							}

							// parent selection is complete, add selected parents to new event
							for _, parent := range parents {
								e.AddParent(parent.ID())
								if e.Lamport() <= parent.Lamport() {
									e.SetLamport(parent.Lamport() + 1)
								}
							}
						}
						// name and ID the event
						e.SetEpoch(1) // use epoch 1 for simulation
						e.Name = fmt.Sprintf("%03d%04d", self, e.Seq())
						hasher := sha256.New()
						hasher.Write(e.Bytes())
						var id [24]byte
						copy(id[:], hasher.Sum(nil)[:24])
						e.SetID(id)
						consensus.SetEventName(e.ID(), fmt.Sprintf("%03d%04d", self, e.Seq()))
						e.creationTime = simTime

						createRandEvent := randEvRNG[self].Float64() < randEvRate // used for introducing randomly created events
						if online[selfID] == true {
							// self is online
							passedTime := simTime - selfParent[self].creationTime
							if passedTime > minEventCreationInterval[self] {
								metric := fcIndexers[self].ValidatorsPastMe()
								if metric < validators.Quorum() {
									metric /= 20
								}
								passedTimeEff := int((uint64(passedTime) * uint64(metric)) / uint64(validators.TotalWeight()))
								if createRandEvent || isLeaf[self] || passedTimeEff > 350 {
									//println(uint64(metric) * 100 / uint64(validators.TotalWeight()))
									//create an event if (i)a random event is created (ii) is a leaf event, or (iii) event timing condition is met
									isLeaf[self] = false // only create one leaf event
									//now start propagation of event to other nodes
									var delay int
									for receiveNode := 0; receiveNode < numValidators; receiveNode++ {
										if receiveNode == self {
											delay = 1 // no delay to send to self (self will 'recieve' its own event after time increment at the top of the main loop)
										} else {
											delay = latency.latency(self, receiveNode, latencyRNG[self])
											// check delay is within min and max bounds
											if delay < 1 {
												delay = 1
											}
											if delay >= maxLatency {
												delay = maxLatency - 1
											}
										}
										receiveTime := (timeIdx + delay) % maxLatency // time index for the circular buffer
										mutex.Lock()
										eventPropagation[receiveTime][self][receiveNode] = append(eventPropagation[receiveTime][self][receiveNode], e) // add the event to the buffer
										mutex.Unlock()
									}

									eventsComplete[self]++ // increment count of events created for this node
									selfParent[self] = *e  //update self parent to be this new event
								}

							}
						}
					}
				}(self)
			}
		}
		wg.Wait()

	}

	// print some useful output
	fmt.Println("")
	// fmt.Println("Simulated time ", float64(simTime)/1000.0, " seconds")
	// fmt.Println("Number of nodes: ", numValidators)
	numOnlineNodes := 0
	for _, isOnline := range online {
		if isOnline {
			numOnlineNodes++
		}
	}
	// fmt.Println("Number of nodes online: ", numOnlineNodes)
	// fmt.Println("Max Total Parents: ", QIParentCount+randParentCount, " Max QI Parents:", QIParentCount, " Max Random Parents", randParentCount)

	// print number of events created by each node
	var totalEventsComplete int = 0
	for _, nEv := range eventsComplete {
		totalEventsComplete += nEv
		// fmt.Println("Stake: ", weights[i], "event rate: ", float64(nEv)*1000/float64(simTime), " events/stake: ", float64(nEv)/float64(weights[i]))
	}
	var maxFrame consensus.Frame = 0
	for _, events := range headsAll {
		for _, event := range events {
			if event.Frame() > maxFrame {
				maxFrame = event.Frame()
			}
		}
	}

	fmt.Println("Max Frame: ", maxFrame)
	// fmt.Println("[Indicator of TTF] Frames per second: ", (1000.0*float64(maxFrame))/float64(simTime))
	fmt.Println(" Number of Events: ", totalEventsComplete)

	fmt.Println("Event rate per (online) node: ", float64(totalEventsComplete)/float64(numOnlineNodes)/(float64(simTime)/1000.0))
	// fmt.Println("[Indictor of gas efficiency] Average events per frame per (online) node: ", (float64(totalEventsComplete))/(float64(maxFrame)*float64(numOnlineNodes)))

	var results Results
	results.maxFrame = maxFrame
	results.numEvents = totalEventsComplete
	return results

}

func updateHeads(newEvent consensus.Event, heads *consensus.Events) {
	// remove newEvent's parents from heads
	for _, parent := range newEvent.Parents() {
		for i := 0; i < len(*heads); i++ {
			if (*heads)[i].ID() == parent {
				(*heads)[i] = (*heads)[len(*heads)-1]
				*heads = (*heads)[:len(*heads)-1]
				// break
			}
		}
	}
	*heads = append(*heads, newEvent) //add newEvent to heads
}

func processEvent(input consensustest.TestEventSource, lchs *consensusengine.CoreLachesis, e *QITestEvent, fcIndexer *ancestor.FCIndexer, heads *consensus.Events, self consensus.ValidatorID, time int) (frame consensus.Frame) {
	input.SetEvent(e)

	_ = lchs.DagIndexer.Add(e)
	_ = lchs.Lachesis.Build(e)
	_ = lchs.Lachesis.Process(e)

	lchs.DagIndexer.Flush()
	// HighestBefore based fc indexer needs to process the event
	fcIndexer.ProcessEvent(&e.BaseEvent)

	updateHeads(e, heads)
	return e.Frame()
}

func (lat *gaussianLatency) latency(sender, receiver int, rng *rand.Rand) int {
	return int(rng.NormFloat64()*lat.std + lat.mean)
}
func sampleDist(rng *rand.Rand, cumDist []float64) (sample int) {
	// generates a random sample from the distribution used to calculate cumDist (using inverse transform sampling)
	random := rng.Float64()
	for sample = 1; cumDist[sample] <= random && sample < len(cumDist); sample++ {
	}
	if sample <= 0 {
		sample = 1 // the distributions used here should not be negative or zero, an explicit check
		fmt.Println("")
		fmt.Println("WARNING: distribution sample was <=0, and reset to 1")
	}
	return sample
}

func stakeCumDist() (cumDist []float64) {
	// the purpose of this function is to calculate a cumulative distribution of validator stake for use in creating random samples from the data distribution

	//list of validator stakes in July 2022
	stakeData := [...]float64{198081564.62, 170755849.45, 145995219.17, 136839786.82, 69530006.55, 40463200.25, 39124627.82, 32452971, 29814402.94, 29171276.63, 26284696.12, 25121739.54, 24461049.53, 23823498.37, 22093834.4, 21578984.4, 20799555.11, 19333530.31, 18250949.01, 17773018.94, 17606393.73, 16559031.91, 15950172.21, 12009825.67, 11049478.07, 9419996.86, 9164450.96, 9162745.35, 7822093.53, 7540197.22, 7344958.29, 7215437.9, 6922757.07, 6556643.44, 5510793.7, 5228201.11, 5140257.3, 4076474.17, 3570632.17, 3428553.68, 3256601.94, 3185019, 3119162.23, 3011027.22, 2860160.77, 2164550.78, 1938492.01, 1690762.63, 1629428.73, 1471177.28, 1300562.06, 1237812.75, 1199822.32, 1095856.64, 1042099.38, 1020613.06, 1020055.55, 946528.43, 863022.57, 826015.44, 800010, 730537, 623529.61, 542996.04, 538920.36, 536288, 519803.37, 505401, 502231, 500100, 500001, 500000}
	stakeDataInt := make([]int, len(stakeData))
	// find the maximum stake in the data
	maxStake := 0
	for i, stake := range stakeData {
		stakeDataInt[i] = int(stake)
		if int(stake) > maxStake {
			maxStake = int(stake)
		}
	}
	// calculate the distribution of the data by dividing into bins
	binVals := make([]float64, maxStake+1)
	for _, stake := range stakeDataInt {
		binVals[stake]++
	}

	//now calculate the cumulative distribution of the delay data
	cumDist = make([]float64, len(binVals))
	npts := float64(len(stakeDataInt))
	cumDist[0] = float64(binVals[0]) / npts
	for i := 1; i < len(cumDist); i++ {
		cumDist[i] = cumDist[i-1] + binVals[i]/npts
	}
	return cumDist
}
