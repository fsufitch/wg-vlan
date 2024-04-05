package main

import (
	"errors"
	"math/big"
	"net"
)

func ipAdd(ip net.IP, offset int64) net.IP {
	return net.IP(big.NewInt(0).Add(big.NewInt(offset), big.NewInt(0).SetBytes(ip)).Bytes())
}

func ipNetEdges(subnet net.IPNet) (net.IP, net.IP) {
	subnetInt := big.NewInt(0).SetBytes(subnet.IP)
	maskInt := big.NewInt(0).SetBytes(subnet.Mask)
	inverseMask := big.NewInt(0).Not(maskInt)
	firstIPInt := big.NewInt(0).And(subnetInt, maskInt)
	lastIPInt := big.NewInt(0).Add(subnetInt, inverseMask)
	return net.IP(firstIPInt.Bytes()), net.IP(lastIPInt.Bytes())
}

func ipCompare(first net.IP, second net.IP) int {
	firstInt := big.NewInt(0).SetBytes(first)
	secondInt := big.NewInt(0).SetBytes(second)
	return firstInt.Cmp(secondInt)
}

func pickNextIP(subnet net.IPNet, takenIPs []net.IP, takenSubnets []net.IPNet) (*net.IP, error) {
	takenIPMap := map[string]struct{}{}
	for _, ip := range takenIPs {
		takenIPMap[ip.String()] = struct{}{}
	}

	startIP, endIP := ipNetEdges(subnet)
	currentIP := ipAdd(startIP, 1)
	for ipCompare(currentIP, endIP) <= 0 {
		if _, ok := takenIPMap[currentIP.String()]; ok {
			// current IP is taken, try the next
			currentIP = ipAdd(currentIP, 1)
			continue
		}

		var takenSubnet *net.IPNet
		for _, subnet := range takenSubnets {
			if subnet.Contains(currentIP) {
				takenSubnet = &subnet
				break
			}
		}

		if takenSubnet != nil {
			_, lastTakenIP := ipNetEdges(subnet)
			currentIP = ipAdd(lastTakenIP, 1)
			continue
		}

		// Passed the conditions, the IP is not taken!
		return &currentIP, nil
	}

	return nil, errors.New("no IP available")
}
