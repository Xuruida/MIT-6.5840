package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"

	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.5840/labgob"
	"6.5840/labrpc"
)

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

const (
	StateFollower int = iota
	StateCandidate
	StateLeader
)

const (
	HeartbeatInterval = 100
	HeartbeatTimeout  = 2000
)

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	currentTerm int
	votedFor    int
	log         []int

	commitIndex int
	lastApplied int

	nextIndex  []int
	matchIndex []int

	state int

	// follower
	receivedHeartBeat bool
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (2A).
	term = rf.currentTerm
	isleader = (rf.state == StateLeader)

	return term, isleader
}

func (rf *Raft) updateTerm(term int) (updated bool) {
	// rf.mu.Lock()
	// defer rf.mu.Unlock()
	rf.PrintStatus()
	if term > rf.currentTerm {
		updated = true
		rf.votedFor = -1
		rf.currentTerm = term
		rf.state = StateFollower
	}
	rf.PrintStatus()
	return
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
// before you've implemented snapshots, you should pass nil as the
// second argument to persister.Save().
// after you've implemented snapshots, pass the current snapshot
// (or nil if there's not yet a snapshot).
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// raftstate := w.Bytes()
	// rf.persister.Save(raftstate, nil)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).
}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGranted bool
}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.updateTerm(args.Term)

	reply.Term = rf.currentTerm
	if args.Term < rf.currentTerm {
		reply.VoteGranted = false
		return
	}

	if rf.votedFor == -1 || rf.votedFor == args.CandidateId {
		rf.votedFor = args.CandidateId
		reply.VoteGranted = true
		rf.receivedHeartBeat = true
		return
	}
}

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) (success bool) {
	// fmt.Println("Sending RequestVote From", rf.me, "to", server)
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	// fmt.Println("Sending RequestVote From", rf.me, "to", server, "Result =", ok)
	return ok
}

func (rf *Raft) PrintStatus() {
	// fmt.Printf("Id: %v, State: %v, currentTerm: %v\n", rf.me, rf.state, rf.currentTerm)
}

func (rf *Raft) Vote() (success bool) {
	// fmt.Printf("id %v: Trying to vote for term(%v)\n", rf.me, rf.currentTerm)
	count := int32(1)
	majority := int32(len(rf.peers)/2 + 1)
	rf.votedFor = rf.me
	stopCh := make(chan struct{})
	chans := make(chan struct{}, len(rf.peers))
	failed := false

	send := func(i int) {
		req := &RequestVoteArgs{
			Term:         rf.currentTerm,
			CandidateId:  rf.me,
			LastLogIndex: 0,
			LastLogTerm:  0,
		}
		resp := &RequestVoteReply{}
		ok := rf.sendRequestVote(i, req, resp)
		if !ok {
			return
		}
		if rf.updateTerm(resp.Term) {
			failed = true
			return
		}
		if resp.VoteGranted {
			chans <- struct{}{}
		}
	}

	for i := range rf.peers {
		if rf.state != StateCandidate {
			return
		}

		if i == rf.me {
			continue
		}

		go send(i)
	}

	go func() {
		rf.sleepElectionTimeout()
		stopCh <- struct{}{}
	}()

	for {
		select {
		case <-stopCh:
			// fmt.Println("Vote reached timeout!")
			return false
		case <-chans:
			atomic.AddInt32(&count, 1)
			if failed {
				return false
			}
			if count >= majority {
				// fmt.Printf("id %v [Vote]: Got %v votes, success = %v\n", rf.me, count, success)
				return true
			}
		}
	}
}

// AppendEntries
type AppendEntriesArgs struct {
	Term     int
	LeaderId int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

func (rf *Raft) RequestAppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	// rf.mu.Lock()
	// defer rf.mu.Unlock()
	rf.updateTerm(args.Term)

	reply.Term = rf.currentTerm
	if args.Term < rf.currentTerm {
		reply.Success = false
	} else {
		reply.Success = true
		rf.receivedHeartBeat = true
	}

}

func (rf *Raft) sendRequestAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.RequestAppendEntries", args, reply)
	return ok
}

// Broadcast heartbeat
func (rf *Raft) BroadcastHeartbeat() bool {
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})

	var wg sync.WaitGroup
	sendHeartbeat := func(target int) {
		req := &AppendEntriesArgs{
			Term:     rf.currentTerm,
			LeaderId: rf.me,
		}
		resp := &AppendEntriesReply{}
		if rf.state == StateLeader {
			ok := rf.sendRequestAppendEntries(target, req, resp)
			if ok && rf.updateTerm(resp.Term) {
				stopCh <- struct{}{}
			}
		} else {
			stopCh <- struct{}{}
		}
		wg.Done()
	}

	for i := range rf.peers {
		if i == rf.me {
			continue
		}
		wg.Add(1)
		go sendHeartbeat(i)
	}

	go func() {
		wg.Wait()
		doneCh <- struct{}{}
	}()

	for {
		select {
		case <-stopCh:
			return false
		case <-doneCh:
			return true
		case <-time.After(HeartbeatTimeout * time.Millisecond):
			return true
		}
	}
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).

	return index, term, isLeader
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) sleepElectionTimeout() {
	et := 200 + (rand.Int63() % 500)
	time.Sleep(time.Duration(et) * time.Millisecond)
}

func (rf *Raft) sleepHeartbeatTimeout() {
	time.Sleep(time.Duration(int64(HeartbeatInterval)) * time.Millisecond)
}

func (rf *Raft) ticker() {
	for rf.killed() == false {

		// Your code here (2A)
		// Check if a leader election should be started.

		// pause for a random amount of time between 50 and 350
		// milliseconds.
		switch rf.state {
		case StateFollower:
			rf.receivedHeartBeat = false
			rf.sleepElectionTimeout()
			// check if heartbeat
			if !rf.receivedHeartBeat {
				rf.state = StateCandidate
			}

		case StateCandidate:
			rf.currentTerm++
			success := rf.Vote()
			// check heartbeat
			if rf.receivedHeartBeat {
				rf.state = StateFollower
			}
			if success && rf.state == StateCandidate {
				rf.state = StateLeader
				fmt.Printf("===== Node %v becomes leader! ====== currentTerm: %v\n", rf.me, rf.currentTerm)
			}

		case StateLeader:
			rf.BroadcastHeartbeat()
			rf.sleepHeartbeatTimeout()

		default:
			// fmt.Printf("Unknown State: %v", rf.state)
			return
		}
	}
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	rf.votedFor = -1
	// Your initialization code here (2A, 2B, 2C).

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}
