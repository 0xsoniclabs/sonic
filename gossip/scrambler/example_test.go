package scrambler

import (
	"reflect"
	"testing"
)

func TestSortPartition_AuthorizationsAreRespectedInOrdering(t *testing.T) {
	var (
		first  = tx(a(0, 0), a(2, 0))
		second = tx(a(1, 0), a(2, 1))
	)
	transactions := []transaction{second, first}

	got := sortPartition(transactions, pickOptimal)
	want := []transaction{first, second}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSortPartition_TransitiveDependency(t *testing.T) {
	var (
		first  = tx(a(0, 0), a(1, 0))
		second = tx(a(1, 1), a(2, 0))
		third  = tx(a(2, 1), a(3, 0))
		fourth = tx(a(3, 1))
	)
	transactions := []transaction{third, second, fourth, first}

	got := sortPartition(transactions, pickOptimal)
	want := []transaction{first, second, third, fourth}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSortPartition_CyclicDependencyWithExecutableSubset(t *testing.T) {
	// T1 -> T2 -> T3 -> T4
	// T4 -> T3
	// ensure T1 and T2 are still sorted correctly
	var (
		first  = tx(a(0, 0), a(1, 0))
		second = tx(a(1, 1), a(2, 0))
		third  = tx(a(2, 1), a(3, 0), a(4, 1))
		fourth = tx(a(3, 1), a(4, 0))
	)
	transactions := []transaction{third, second, fourth, first}

	got := sortPartition(transactions, pickOptimal)
	want := []transaction{first, second, third, fourth}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSortPartition_CycleBetweenTransactionSenderAndAuthorization(t *testing.T) {
	var (
		first  = tx(a(1, 0), a(2, 1))
		second = tx(a(1, 1), a(2, 0))
	)
	transactions := []transaction{second, first}

	got := sortPartition(transactions, pickOptimal)
	want := []transaction{first, second}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSortPartition_DependencyBetweenTransactionSenderAndAuthorization(t *testing.T) {
	var (
		first  = tx(a(1, 0), a(2, 0))
		second = tx(a(2, 1), a(1, 1))
	)
	transactions := []transaction{second, first}

	got := sortPartition(transactions, pickOptimal)
	want := []transaction{first, second}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
