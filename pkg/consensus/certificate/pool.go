package certificate

import (
	"sync"
)

type Pool struct {
	nonGossiped SingleCommits
	gossiped    SingleCommits
	mutex       *sync.Mutex
}

func NewPool() *Pool {
	return &Pool{
		nonGossiped: SingleCommits{},
		gossiped:    SingleCommits{},
		mutex:       new(sync.Mutex),
	}
}

func (p *Pool) Size() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return len(p.gossiped) + len(p.nonGossiped)
}

func (p *Pool) Has(val *SingleCommit) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.gossiped.has(val) || p.nonGossiped.has(val)
}

func (p *Pool) Add(val *SingleCommit) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.nonGossiped = append(p.nonGossiped, val)
}

func (p *Pool) Cleanup(checker func(h uint32) bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	updatedGossip := SingleCommits{}
	for _, commit := range p.gossiped {
		if checker(commit.height) {
			updatedGossip = append(updatedGossip, commit)
		}
	}
	p.gossiped = updatedGossip
	updatedNonGossip := SingleCommits{}
	for _, commit := range p.nonGossiped {
		if checker(commit.height) {
			updatedNonGossip = append(updatedNonGossip, commit)
		}
	}
	p.nonGossiped = updatedNonGossip
}

func (p *Pool) Select(maxHeightPrecommited uint32, limit int) SingleCommits {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	max := uint32(0)
	if maxHeightPrecommited > CommitRangeStored {
		max = maxHeightPrecommited - CommitRangeStored
	}
	result := SingleCommits{}
	p.nonGossiped.Sort()
	commits := p.nonGossiped.GetUntil(max)
	result = append(result, commits...)
	if len(result) >= limit {
		return result[:limit]
	}
	p.gossiped.Sort()
	commits = p.gossiped.GetUntil(max)
	result = append(result, commits...)
	if len(result) >= limit {
		return result[:limit]
	}
	commits = p.nonGossiped.GetLargestWithLimit(limit-len(result), true)
	result = append(result, commits...)
	if len(result) >= limit {
		return result[:limit]
	}
	commits = p.nonGossiped.GetLargestWithLimit(limit-len(result), false)
	result = append(result, commits...)
	if len(result) >= limit {
		return result[:limit]
	}
	return result
}

func (p *Pool) Get(height uint32) SingleCommits {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	// return all single commits by height
	commits := []*SingleCommit{}
	for _, sc := range p.gossiped {
		if sc.height == height {
			commits = append(commits, sc)
		}
	}
	for _, sc := range p.nonGossiped {
		if sc.height == height {
			commits = append(commits, sc)
		}
	}
	return commits
}

func (p *Pool) Upgrade(commits SingleCommits) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	gossiped := SingleCommits{}
	remaining := SingleCommits{}
	for _, commit := range p.nonGossiped {
		if commits.has(commit) {
			gossiped = append(gossiped, commit)
			continue
		}
		remaining = append(remaining, commit)
	}
	p.nonGossiped = remaining
	p.gossiped = append(p.gossiped, gossiped...)
}
