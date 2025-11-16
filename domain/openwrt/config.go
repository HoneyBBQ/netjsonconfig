package openwrt

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	devicev1 "github.com/honeybbq/netjson/gen/go/netjson/device/v1"
	openvpnv1 "github.com/honeybbq/netjson/gen/go/netjson/openvpn/v1"
	openwrtv1 "github.com/honeybbq/netjson/gen/go/netjson/openwrt/v1"
	zerotierv1 "github.com/honeybbq/netjson/gen/go/netjson/zerotier/v1"

	helpers "github.com/honeybbq/netjsonconfig/domain/utils"
	"github.com/honeybbq/netjsonconfig/pkg/ast/uci"
	"github.com/honeybbq/netjsonconfig/pkg/nxerrors"
)

// Config 表示 OpenWrt 领域模型。
type Config struct {
	Message *openwrtv1.OpenWrtConfig
}

// FromProto 构造领域模型。
func FromProto(msg *openwrtv1.OpenWrtConfig) (*Config, error) {
	if msg == nil {
		return nil, nxerrors.New(nxerrors.KindValidation, fmt.Errorf("config is nil"))
	}
	return &Config{Message: msg}, nil
}

// ToAST 转换为 UCI 文档（最小 AST）。
func (c *Config) ToAST() (*uci.Document, error) {
	if c == nil {
		return nil, nxerrors.New(nxerrors.KindInternal, errors.New("config is nil"))
	}
	if c.Message == nil {
		return nil, nxerrors.New(nxerrors.KindInternal, errors.New("openwrt message is nil"))
	}

	var packages []*uci.Package
	if pkg := buildSystemPackage(c.Message); pkg != nil {
		packages = append(packages, pkg)
	}
	if pkg := buildWirelessPackage(c.Message); pkg != nil {
		packages = append(packages, pkg)
	}
	if pkg := buildNetworkPackage(c.Message); pkg != nil {
		packages = append(packages, pkg)
	}
	if pkg := buildOpenvpnPackage(c.Message); pkg != nil {
		packages = append(packages, pkg)
	}
	if pkg := buildZerotierPackage(c.Message); pkg != nil {
		packages = append(packages, pkg)
	}

	if len(packages) == 0 {
		return nil, nxerrors.New(nxerrors.KindRender, fmt.Errorf("no supported netjson fields found"))
	}

	return &uci.Document{
		Packages: packages,
		Files:    c.Message.GetFiles(),
	}, nil
}

func buildSystemPackage(msg *openwrtv1.OpenWrtConfig) *uci.Package {
	if msg == nil {
		return nil
	}

	var sections []*uci.Section
	if general := buildGeneralSection(msg.GetGeneral()); general != nil {
		sections = append(sections, general)
	}
	if ntp := buildNtpSection(msg.GetNtp()); ntp != nil {
		sections = append(sections, ntp)
	}
	sections = append(sections, buildLedSections(msg.GetLeds())...)

	if len(sections) == 0 {
		return nil
	}

	return &uci.Package{
		Name:     "system",
		Sections: sections,
	}
}

func buildNetworkPackage(msg *openwrtv1.OpenWrtConfig) *uci.Package {
	if msg == nil {
		return nil
	}

	var sections []*uci.Section
	if globals := buildGlobalsSection(msg.GetGeneral()); globals != nil {
		sections = append(sections, globals)
	}
	sections = append(sections, buildSwitchSections(msg.GetSwitches())...)

	// DSA style: build device sections first, then bridge-vlan, then interface sections
	for _, iface := range msg.GetInterfaces() {
		if iface.GetWireless() != nil {
			continue
		}
		// Build device section for bridges (DSA >= 21)
		if deviceSec := buildDeviceSection(iface); deviceSec != nil {
			sections = append(sections, deviceSec)
		}
	}

	// Build bridge-vlan sections for VLAN filtering
	for _, iface := range msg.GetInterfaces() {
		if iface.GetWireless() != nil {
			continue
		}
		sections = append(sections, buildBridgeVlanSections(iface)...)
	}

	// Then build interface sections
	for _, iface := range msg.GetInterfaces() {
		if iface.GetWireless() != nil {
			continue
		}
		if section := buildInterfaceSection(iface, msg); section != nil {
			sections = append(sections, section)
		}
	}
	sections = append(sections, buildWireguardPeerSections(msg.GetWireguardPeers())...)
	sections = append(sections, buildRouteSections(msg.GetRoutes())...)
	sections = append(sections, buildRuleSections(msg.GetIpRules())...)

	if len(sections) == 0 {
		return nil
	}

	return &uci.Package{
		Name:     "network",
		Sections: sections,
	}
}

func buildGlobalsSection(general *devicev1.General) *uci.Section {
	if general == nil || general.GetUlaPrefix() == "" {
		return nil
	}
	name := "globals"
	if general.GetGlobalsId() != "" {
		name = general.GetGlobalsId()
	}
	section := uci.NewSection("globals", name)
	helpers.SetString(section, "ula_prefix", general.GetUlaPrefix())
	return section
}

// buildDeviceSection creates device section for DSA style (OpenWrt >= 21).
// Only bridges need device sections; other types handled in interface section.
func buildDeviceSection(iface *devicev1.Interface) *uci.Section {
	if iface == nil {
		return nil
	}

	isBridge := strings.EqualFold(iface.GetType(), "bridge")
	if !isBridge {
		return nil
	}

	// Generate device name: device_xxx
	deviceName := fmt.Sprintf("device_%s", iface.GetName())
	section := uci.NewSection("device", deviceName)

	// Bridge device name: br-xxx
	bridgeName := fmt.Sprintf("br-%s", iface.GetName())
	helpers.SetString(section, "name", bridgeName)
	helpers.SetString(section, "type", "bridge")

	// Bridge members go to device section as ports
	for _, member := range iface.GetBridgeMembers() {
		helpers.AppendList(section, "ports", member)
	}

	// MTU goes to device section
	helpers.SetUint32Ptr(section, "mtu", iface.Mtu)

	// Bridge-specific L2 options (from Interface directly)
	helpers.SetBool(section, "stp", iface.Stp)
	helpers.SetBool(section, "igmp_snooping", iface.IgmpSnooping)

	// Enable VLAN filtering if configured
	if len(iface.GetVlanFiltering()) > 0 {
		helpers.SetBoolValue(section, "vlan_filtering", true)
	}

	// Bridge-specific L2 options (from BridgeSettings if present)
	if bridge := iface.GetBridge(); bridge != nil {
		helpers.SetUint32Ptr(section, "forward_delay", bridge.ForwardDelay)
		helpers.SetUint32Ptr(section, "hello_time", bridge.HelloTime)
		helpers.SetUint32Ptr(section, "priority", bridge.Priority)
		helpers.SetUint32Ptr(section, "ageing_time", bridge.AgeingTime)
		helpers.SetUint32Ptr(section, "max_age", bridge.MaxAge)
		helpers.SetBool(section, "multicast_querier", bridge.MulticastQuerier)
		helpers.SetUint32Ptr(section, "query_interval", bridge.QueryInterval)
		helpers.SetUint32Ptr(section, "query_response_interval", bridge.QueryResponseInterval)
		helpers.SetUint32Ptr(section, "last_member_interval", bridge.LastMemberInterval)
		helpers.SetUint32Ptr(section, "hash_max", bridge.HashMax)
		helpers.SetUint32Ptr(section, "robustness", bridge.Robustness)
	}

	if len(section.Options) == 0 && len(section.Lists) == 0 {
		return nil
	}

	return section
}

// buildBridgeVlanSections creates bridge-vlan sections for VLAN filtering (DSA).
// Returns both bridge-vlan sections and corresponding interface sections.
func buildBridgeVlanSections(iface *devicev1.Interface) []*uci.Section {
	if iface == nil || len(iface.GetVlanFiltering()) == 0 {
		return nil
	}

	isBridge := strings.EqualFold(iface.GetType(), "bridge")
	if !isBridge {
		return nil
	}

	var bridgeVlanSections []*uci.Section
	var vlanInterfaces []*uci.Section
	bridgeName := fmt.Sprintf("br-%s", iface.GetName())

	for _, vlan := range iface.GetVlanFiltering() {
		if vlan == nil {
			continue
		}

		vlanID := vlan.GetVlan()
		if vlanID == 0 {
			continue
		}

		// Generate bridge-vlan section name: vlan_xxx_100
		sectionName := fmt.Sprintf("vlan_%s_%d", iface.GetName(), vlanID)
		section := uci.NewSection("bridge-vlan", sectionName)

		helpers.SetString(section, "device", bridgeName)
		helpers.SetUint32Value(section, "vlan", vlanID)

		// Build ports list with tagging
		for _, port := range vlan.GetPorts() {
			if port == nil || port.GetIfname() == "" {
				continue
			}
			// Format: "eth0:t" or "eth1:u" (tagged/untagged)
			portStr := fmt.Sprintf("%s:%s", port.GetIfname(), port.GetTagging())
			if port.GetPrimaryVid() {
				portStr += "*" // Primary VID marker
			}
			helpers.AppendList(section, "ports", portStr)
		}

		if len(section.Options) > 0 || len(section.Lists) > 0 {
			bridgeVlanSections = append(bridgeVlanSections, section)
		}

		// Create corresponding interface for VLAN (proto=none)
		vlanIfaceName := fmt.Sprintf("%s_%d", iface.GetName(), vlanID)
		vlanIface := uci.NewSection("interface", vlanIfaceName)
		vlanDevice := fmt.Sprintf("%s.%d", bridgeName, vlanID)
		helpers.SetString(vlanIface, "device", vlanDevice)
		helpers.SetString(vlanIface, "proto", "none")
		vlanInterfaces = append(vlanInterfaces, vlanIface)
	}

	// Return bridge-vlan sections first, then interface sections
	return append(bridgeVlanSections, vlanInterfaces...)
}

func buildInterfaceSection(iface *devicev1.Interface, msg *openwrtv1.OpenWrtConfig) *uci.Section {
	if iface == nil || iface.GetName() == "" {
		return nil
	}
	section := uci.NewSection("interface", iface.GetName())
	isWireguard := strings.EqualFold(iface.GetType(), "wireguard")
	isBridge := strings.EqualFold(iface.GetType(), "bridge")

	// DSA style: set device option
	if isBridge {
		// Bridge references the device section: br-xxx
		bridgeName := fmt.Sprintf("br-%s", iface.GetName())
		helpers.SetString(section, "device", bridgeName)
	} else if !isWireguard {
		// Non-bridge, non-wireguard: use device name or interface name
		device := iface.GetDevice()
		if device == "" {
			device = iface.GetName()
		}
		helpers.SetString(section, "device", device)
	} else {
		// Wireguard: set device if provided
		helpers.SetString(section, "device", iface.GetDevice())
	}

	// Wireguard needs type option
	if isWireguard {
		helpers.SetString(section, "type", "wireguard")
	}
	helpers.SetStringPtr(section, "ip4table", iface.Ip4Table)
	helpers.SetStringPtr(section, "ip6table", iface.Ip6Table)
	helpers.SetStringPtr(section, "ip6hint", iface.Ip6Hint)
	helpers.SetStringPtr(section, "ip6ifaceid", iface.Ip6Ifaceid)
	helpers.SetStringPtr(section, "ip6gw", iface.Ip6Gateway)
	helpers.SetStringPtr(section, "zone", iface.FirewallZone)
	// MTU for bridges goes to device section (DSA), only set here for non-bridges
	if !isBridge {
		helpers.SetUint32Ptr(section, "mtu", iface.Mtu)
	}
	helpers.SetUint32Ptr(section, "metric", iface.Metric)
	helpers.SetUint32Ptr(section, "txqueuelen", iface.Txqueuelen)
	helpers.SetBool(section, "disabled", iface.Disabled)
	helpers.SetBool(section, "auto", iface.Autostart)
	helpers.SetBool(section, "force_link", iface.ForceLink)
	helpers.SetBool(section, "delegate", iface.Delegate)
	helpers.SetBool(section, "ipv6", iface.Ipv6)
	helpers.SetBool(section, "peerdns", iface.PeerDns)
	helpers.SetBool(section, "defaultroute", iface.DefaultRoute)
	helpers.SetBool(section, "broadcast", iface.Broadcast)
	helpers.SetBool(section, "sourcefilter", iface.SourceFilter)
	helpers.SetString(section, "fwmark", iface.GetFwmark())

	if iface.Mac != nil && *iface.Mac != "" {
		helpers.SetString(section, "macaddr", *iface.Mac)
	}

	// DSA: bridge members go to device section as ports, not here
	// Only handle non-bridge ifname
	if !isBridge && len(iface.Ifname) > 0 {
		for _, name := range iface.Ifname {
			helpers.AppendList(section, "ifname", name)
		}
	}

	if proto := iface.GetProto(); proto != "" {
		helpers.SetString(section, "proto", proto)
	}
	if isWireguard {
		helpers.SetString(section, "proto", "wireguard")
	}

	applyInterfaceAddresses(section, iface, isWireguard, isBridge)
	applyDNS(section, iface, msg)
	if isWireguard {
		applyWireguardInterface(section, iface)
	}

	if len(section.Options) == 0 && len(section.Lists) == 0 {
		return nil
	}
	return section
}

func applyInterfaceAddresses(section *uci.Section, iface *devicev1.Interface, isWireguard, isBridge bool) {
	ifaceProtoSet := helpers.OptionExists(section, "proto")
	for _, addr := range iface.GetAddresses() {
		family := addr.GetFamily()
		switch family {
		case "ipv4", "":
			if !ifaceProtoSet && addr.GetProto() != "" {
				helpers.SetString(section, "proto", addr.GetProto())
				ifaceProtoSet = true
			}
			if addr.GetProto() == "dhcp" {
				continue
			}
			if addr.GetAddress() != "" {
				if isWireguard {
					value := addr.GetAddress()
					if mask := addr.GetMask(); mask != 0 {
						value = fmt.Sprintf("%s/%d", value, mask)
					}
					helpers.AppendList(section, "addresses", value)
				} else {
					// DSA style: all interfaces use option ipaddr (not list)
					helpers.SetString(section, "ipaddr", addr.GetAddress())
					if mask := addr.GetMask(); mask != 0 {
						if netmask := prefixToNetmask(mask); netmask != "" {
							helpers.SetString(section, "netmask", netmask)
						}
					}
				}
			}
			if !isWireguard {
				if gateway := addr.GetGateway(); gateway != "" {
					helpers.SetString(section, "gateway", gateway)
				}
			}
		case "ipv6":
			if !ifaceProtoSet && addr.GetProto() != "" {
				helpers.SetString(section, "proto", addr.GetProto())
				ifaceProtoSet = true
			}
			if addr.GetProto() == "dhcpv6" {
				continue
			}
			if addr.GetAddress() != "" {
				value := addr.GetAddress()
				if mask := addr.GetMask(); mask != 0 {
					value = fmt.Sprintf("%s/%d", value, mask)
				}
				if isWireguard {
					helpers.AppendList(section, "addresses", value)
				} else {
					helpers.AppendList(section, "ip6addr", value)
				}
			}
		}
	}
}

func applyDNS(section *uci.Section, iface *devicev1.Interface, msg *openwrtv1.OpenWrtConfig) {
	// Interface-level DNS takes precedence
	ifaceDns := iface.GetDns()
	ifaceDnsSearch := iface.GetDnsSearch()

	// Check if proto should ignore global DNS
	proto := section.Options["proto"]
	ignoreGlobalDNS := false
	if len(proto) > 0 {
		p := proto[0]
		ignoreGlobalDNS = p == "dhcp" || p == "dhcpv6" || p == "none"
	}

	// Apply DNS servers (space-separated string, not list)
	var dnsServers []string
	if len(ifaceDns) > 0 {
		dnsServers = ifaceDns
	} else if !ignoreGlobalDNS && msg != nil && len(msg.GetDnsServers()) > 0 {
		dnsServers = msg.GetDnsServers()
	}
	if len(dnsServers) > 0 {
		helpers.SetString(section, "dns", strings.Join(dnsServers, " "))
	}

	// Apply DNS search domains (space-separated string, not list)
	var dnsSearch []string
	if len(ifaceDnsSearch) > 0 {
		dnsSearch = ifaceDnsSearch
	} else if !ignoreGlobalDNS && msg != nil && len(msg.GetDnsSearch()) > 0 {
		dnsSearch = msg.GetDnsSearch()
	}
	if len(dnsSearch) > 0 {
		helpers.SetString(section, "dns_search", strings.Join(dnsSearch, " "))
	}
}

func prefixToNetmask(prefix uint32) string {
	if prefix > 32 {
		return ""
	}
	mask := net.CIDRMask(int(prefix), 32)
	if mask == nil {
		return ""
	}
	return net.IP(mask).String()
}

func applyWireguardInterface(section *uci.Section, iface *devicev1.Interface) {
	wg := iface.GetWireguard()
	if wg != nil {
		helpers.SetString(section, "private_key", wg.GetPrivateKey())
		helpers.SetString(section, "public_key", wg.GetPublicKey())
		helpers.SetUint32Ptr(section, "listen_port", wg.ListenPort)
		for _, addr := range wg.GetAddresses() {
			if addr != "" {
				helpers.AppendList(section, "addresses", addr)
			}
		}
	}
	helpers.SetBool(section, "nohostroute", iface.NoHostRoute)
	helpers.SetString(section, "fwmark", iface.GetFwmark())
}

func buildRouteSections(routes []*devicev1.StaticRoute) []*uci.Section {
	var sections []*uci.Section
	if len(routes) == 0 {
		return sections
	}

	counter := 1
	for _, route := range routes {
		if route == nil {
			continue
		}
		dest := route.GetDestination()
		isIPv6 := strings.Contains(dest, ":")
		sectionType := "route"
		if isIPv6 {
			sectionType = "route6"
		}
		name := route.GetName()
		if name == "" {
			name = fmt.Sprintf("route%d", counter)
		}
		counter++

		section := uci.NewSection(sectionType, name)
		helpers.SetString(section, "interface", route.GetDevice())
		helpers.SetString(section, "gateway", route.GetNext())
		helpers.SetString(section, "source", route.GetSource())
		helpers.SetString(section, "table", route.GetTable())
		helpers.SetString(section, "type", route.GetType())
		helpers.SetUint32Ptr(section, "metric", route.Cost)
		helpers.SetUint32Ptr(section, "mtu", route.Mtu)
		helpers.SetBool(section, "onlink", route.Onlink)

		if isIPv6 {
			helpers.SetString(section, "target", dest)
		} else {
			target, netmask := splitIPv4Destination(dest)
			helpers.SetString(section, "target", target)
			if netmask != "" {
				helpers.SetString(section, "netmask", netmask)
			}
		}

		sections = append(sections, section)
	}

	return sections
}

func splitIPv4Destination(dest string) (string, string) {
	if dest == "" {
		return "", ""
	}
	parts := strings.Split(dest, "/")
	if len(parts) == 2 {
		if prefix, err := strconv.Atoi(parts[1]); err == nil {
			return parts[0], prefixToNetmask(uint32(prefix))
		}
	}
	return dest, ""
}

func buildRuleSections(rules []*openwrtv1.IpRule) []*uci.Section {
	var sections []*uci.Section
	if len(rules) == 0 {
		return sections
	}

	counter := 1
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		isIPv6 := isIPv6Rule(rule)
		sectionType := "rule"
		if isIPv6 {
			sectionType = "rule6"
		}
		name := rule.GetName()
		if name == "" {
			name = fmt.Sprintf("rule%d", counter)
		}
		counter++

		section := uci.NewSection(sectionType, name)
		helpers.SetString(section, "action", rule.GetAction())
		helpers.SetString(section, "src", rule.GetSrc())
		helpers.SetString(section, "dest", rule.GetDest())
		helpers.SetString(section, "in", rule.GetIn())
		helpers.SetString(section, "out", rule.GetOut())
		helpers.SetString(section, "lookup", rule.GetLookup())
		helpers.SetString(section, "mark", rule.GetMark())
		helpers.SetUint32Ptr(section, "tos", rule.Tos)
		helpers.SetUint32Ptr(section, "goto", rule.Goto)
		helpers.SetBool(section, "invert", rule.Invert)

		sections = append(sections, section)
	}

	return sections
}

func isIPv6Rule(rule *openwrtv1.IpRule) bool {
	if rule == nil {
		return false
	}
	if strings.Contains(rule.GetDest(), ":") || strings.Contains(rule.GetSrc(), ":") {
		return true
	}
	return false
}

func buildWireguardPeerSections(peers []*openwrtv1.WireguardPeerConfig) []*uci.Section {
	var sections []*uci.Section
	if len(peers) == 0 {
		return sections
	}

	counters := make(map[string]int)
	for _, peer := range peers {
		if peer == nil || peer.GetInterface() == "" {
			continue
		}
		iface := sanitizeIdentifier(peer.GetInterface())
		if iface == "" {
			continue
		}
		sectionType := fmt.Sprintf("wireguard_%s", iface)
		index := counters[iface]
		counters[iface]++
		sectionName := fmt.Sprintf("wgpeer_%s", iface)
		if index > 0 {
			sectionName = fmt.Sprintf("%s_%d", sectionName, index+1)
		}

		section := uci.NewSection(sectionType, sectionName)
		helpers.SetList(section, "allowed_ips", peer.GetAllowedIps())
		helpers.SetString(section, "endpoint_host", peer.GetEndpointHost())
		helpers.SetUint32Ptr(section, "endpoint_port", peer.EndpointPort)
		helpers.SetUint32Ptr(section, "persistent_keepalive", peer.PersistentKeepalive)
		helpers.SetString(section, "preshared_key", peer.GetPresharedKey())
		helpers.SetString(section, "public_key", peer.GetPublicKey())
		helpers.SetBool(section, "route_allowed_ips", peer.RouteAllowedIps)
		sections = append(sections, section)
	}

	return sections
}

func buildWirelessPackage(msg *openwrtv1.OpenWrtConfig) *uci.Package {
	if msg == nil {
		return nil
	}

	var sections []*uci.Section
	for _, radio := range msg.GetRadios() {
		if section := buildWifiDeviceSection(radio); section != nil {
			sections = append(sections, section)
		}
	}
	for _, iface := range msg.GetInterfaces() {
		wifi := iface.GetWireless()
		if wifi == nil {
			continue
		}
		if section := buildWifiIfaceSection(iface.GetName(), wifi); section != nil {
			sections = append(sections, section)
		}
	}
	if len(sections) == 0 {
		return nil
	}

	return &uci.Package{
		Name:     "wireless",
		Sections: sections,
	}
}

func buildWifiDeviceSection(radio *devicev1.Radio) *uci.Section {
	if radio == nil || radio.GetName() == "" {
		return nil
	}
	section := uci.NewSection("wifi-device", radio.GetName())

	section.Options["type"] = []string{"mac80211"}
	helpers.SetString(section, "band", radio.GetBand())
	if channel := radio.GetChannel(); channel != 0 {
		helpers.SetUint32Value(section, "channel", channel)
	}
	helpers.SetString(section, "htmode", radio.GetHtmode())
	helpers.SetString(section, "country", radio.GetCountry())
	helpers.SetUint32Value(section, "txpower", radio.GetTxPower())
	helpers.SetBool(section, "disabled", radio.Disabled)

	return section
}

func buildWifiIfaceSection(interfaceName string, wifi *devicev1.WirelessSettings) *uci.Section {
	if wifi == nil {
		return nil
	}

	sectionName := interfaceName
	if sectionName == "" {
		sectionName = wifi.GetRadio()
	}
	if sectionName == "" {
		return nil
	}
	section := uci.NewSection("wifi-iface", fmt.Sprintf("wifi_%s", sectionName))

	helpers.SetString(section, "device", wifi.GetRadio())
	if mode := mapWirelessMode(wifi.GetMode()); mode != "" {
		helpers.SetString(section, "mode", mode)
	}
	helpers.SetString(section, "ssid", wifi.GetSsid())
	helpers.SetString(section, "bssid", wifi.GetBssid())
	helpers.SetBool(section, "hidden", wifi.Hidden)
	helpers.SetBool(section, "wds", wifi.Wds)
	helpers.SetBool(section, "wmm", wifi.Wmm)
	helpers.SetBool(section, "isolate", wifi.Isolate)
	helpers.SetBool(section, "ieee80211r", wifi.Ieee80211R)
	helpers.SetBool(section, "ft_psk_generate_local", wifi.FtPskGenerateLocal)
	helpers.SetBool(section, "ft_over_ds", wifi.FtOverDs)
	helpers.SetBool(section, "rsn_preauth", wifi.RsnPreauth)
	helpers.SetString(section, "macfilter", wifi.GetMacfilter())
	if len(wifi.GetMaclist()) > 0 {
		helpers.SetList(section, "maclist", wifi.GetMaclist())
	}
	helpers.SetString(section, "ifname", interfaceName)

	setWirelessNetwork(section, wifi.GetNetwork())
	applyWirelessEncryption(section, wifi.GetEncryption())

	return section
}

// setWirelessNetwork stores network reference(s).
// Single value → option, multiple → list (OpenWrt convention).
func setWirelessNetwork(section *uci.Section, networks []string) {
	if len(networks) == 0 {
		return
	}
	// Filter empty values
	filtered := make([]string, 0, len(networks))
	for _, n := range networks {
		if n != "" {
			filtered = append(filtered, n)
		}
	}
	if len(filtered) == 0 {
		return
	}
	// Single value uses option, multiple uses list
	if len(filtered) == 1 {
		helpers.SetString(section, "network", filtered[0])
		return
	}
	helpers.SetList(section, "network", filtered)
}

func applyWirelessEncryption(section *uci.Section, enc *devicev1.WirelessEncryption) {
	if enc == nil {
		return
	}
	encryption := mapEncryptionProtocol(enc.GetProtocol())
	if encryption != "" {
		helpers.SetString(section, "encryption", encryption)
	}
	helpers.SetString(section, "cipher", enc.GetCipher())
	helpers.SetString(section, "ieee80211w", enc.GetIeee80211W())
	helpers.SetString(section, "key", enc.GetKey())
	helpers.SetString(section, "server", enc.GetServer())
	helpers.SetUint32Ptr(section, "port", enc.Port)
	helpers.SetString(section, "acct_server", enc.GetAcctServer())
	helpers.SetUint32Ptr(section, "acct_port", enc.AcctServerPort)
	helpers.SetBool(section, "disabled", enc.Disabled)
}

func mapWirelessMode(mode string) string {
	switch strings.ToLower(mode) {
	case "access_point", "ap":
		return "ap"
	case "station", "sta", "client":
		return "sta"
	case "mesh", "802.11s":
		return "mesh"
	case "monitor":
		return "monitor"
	case "adhoc":
		return "adhoc"
	default:
		return mode
	}
}

func mapEncryptionProtocol(proto string) string {
	switch strings.ToLower(proto) {
	case "", "none", "open":
		return "none"
	case "wep":
		return "wep"
	case "wpa_personal", "wpa", "wpa1_personal":
		return "psk"
	case "wpa2_personal", "wpa2":
		return "psk2"
	case "wpa2_personal_mixed", "wpa_personal_mixed":
		return "psk-mixed"
	case "wpa3_personal":
		return "sae"
	case "wpa3_personal_mixed":
		return "sae-mixed"
	default:
		return proto
	}
}

func sanitizeSectionName(prefix, preferred string, idx int) string {
	if clean := sanitizeIdentifier(preferred); clean != "" {
		return clean
	}
	return fmt.Sprintf("%s%d", prefix, idx+1)
}

func sanitizeIdentifier(value string) string {
	if value == "" {
		return ""
	}
	clean := strings.ToLower(value)
	clean = strings.ReplaceAll(clean, " ", "_")
	clean = strings.ReplaceAll(clean, "-", "_")
	clean = strings.ReplaceAll(clean, ".", "_")
	clean = strings.ReplaceAll(clean, "'", "")
	return clean
}

func buildSwitchSections(switches []*openwrtv1.SwitchConfig) []*uci.Section {
	var sections []*uci.Section
	for idx, sw := range switches {
		if sw == nil || sw.GetName() == "" {
			continue
		}
		switchName := sanitizeSectionName("switch", sw.GetName(), idx)
		sec := uci.NewSection("switch", switchName)
		helpers.SetString(sec, "name", sw.GetName())
		if sw.Reset_ != nil {
			helpers.SetBool(sec, "reset", sw.Reset_)
		}
		if sw.EnableVlan != nil {
			helpers.SetBool(sec, "enable_vlan", sw.EnableVlan)
		}
		sections = append(sections, sec)

		for vlanIdx, vlan := range sw.GetVlans() {
			if vlan == nil {
				continue
			}
			vlanName := sanitizeSectionName(fmt.Sprintf("%s_vlan", sw.GetName()), fmt.Sprintf("%s_vlan%d", sw.GetName(), vlanIdx+1), vlanIdx)
			vlanSec := uci.NewSection("switch_vlan", vlanName)
			helpers.SetString(vlanSec, "device", vlan.GetDevice())
			if vlan.GetVlanId() != 0 {
				helpers.SetUint32Value(vlanSec, "vlan", vlan.GetVlanId())
			}
			helpers.SetString(vlanSec, "ports", vlan.GetPorts())
			sections = append(sections, vlanSec)
		}
	}
	return sections
}

func buildGeneralSection(general *devicev1.General) *uci.Section {
	if general == nil {
		return nil
	}
	section := uci.NewSection("system", "system")

	values := helpers.ProtoMessageToMap(general)
	if len(values) == 0 {
		return nil
	}

	// exclude globals-only fields
	delete(values, "ula_prefix")
	delete(values, "globals_id")

	helpers.ApplyOptionsFromMap(section, values, nil)

	if len(section.Options) == 0 && len(section.Lists) == 0 {
		return nil
	}
	return section
}

func buildNtpSection(ntp *openwrtv1.NtpSettings) *uci.Section {
	if ntp == nil {
		return nil
	}
	section := uci.NewSection("timeserver", "ntp")
	helpers.SetBool(section, "enabled", ntp.Enabled)
	helpers.SetBool(section, "enable_server", ntp.EnableServer)
	helpers.SetList(section, "servers", ntp.GetServers())
	helpers.SetStringPtr(section, "hostname", ntp.Hostname)
	helpers.SetUint32Ptr(section, "port", ntp.Port)
	helpers.SetList(section, "pools", ntp.GetPools())

	if len(section.Options) == 0 && len(section.Lists) == 0 {
		return nil
	}
	return section
}

func buildLedSections(leds []*openwrtv1.Led) []*uci.Section {
	var sections []*uci.Section
	for idx, led := range leds {
		if led == nil || led.GetName() == "" {
			continue
		}
		identifier := sanitizeIdentifier(led.GetName())
		if identifier == "" {
			identifier = fmt.Sprintf("led%d", idx+1)
		}
		name := fmt.Sprintf("led_%s", identifier)
		section := uci.NewSection("led", name)
		helpers.SetString(section, "name", led.GetName())
		helpers.SetString(section, "sysfs", led.GetSysfs())
		helpers.SetString(section, "trigger", led.GetTrigger())

		values := helpers.ProtoMessageToMap(led)
		skip := map[string]struct{}{
			"name":    {},
			"sysfs":   {},
			"trigger": {},
		}
		helpers.ApplyOptionsFromMap(section, values, skip)

		sections = append(sections, section)
	}
	return sections
}

func buildOpenvpnPackage(msg *openwrtv1.OpenWrtConfig) *uci.Package {
	if msg == nil || len(msg.GetOpenvpn()) == 0 {
		return nil
	}

	var sections []*uci.Section
	for _, vpn := range msg.GetOpenvpn() {
		if vpn == nil || vpn.GetName() == "" {
			continue
		}
		if section := buildOpenvpnSection(vpn); section != nil {
			sections = append(sections, section)
		}
	}

	if len(sections) == 0 {
		return nil
	}

	return &uci.Package{
		Name:     "openvpn",
		Sections: sections,
	}
}

func buildOpenvpnSection(vpn *openvpnv1.OpenVpnInstance) *uci.Section {
	if vpn == nil {
		return nil
	}

	name := sanitizeIdentifier(vpn.GetName())
	if name == "" {
		return nil
	}

	section := uci.NewSection("openvpn", name)

	// enabled field defaults to true in OpenWrt
	helpers.SetBoolValue(section, "enabled", true)

	// Use ProtoMessageToMap for remaining fields
	values := helpers.ProtoMessageToMap(vpn)
	skip := map[string]struct{}{
		"name": {},
	}
	helpers.ApplyOptionsFromMap(section, values, skip)

	return section
}

func buildZerotierPackage(msg *openwrtv1.OpenWrtConfig) *uci.Package {
	if msg == nil || len(msg.GetZerotier()) == 0 {
		return nil
	}

	var sections []*uci.Section
	for _, zt := range msg.GetZerotier() {
		if zt == nil || zt.GetName() == "" {
			continue
		}
		// Main zerotier section
		if section := buildZerotierSection(zt); section != nil {
			sections = append(sections, section)
		}
		// Network sections (for ZeroTier > 1.14)
		sections = append(sections, buildZerotierNetworkSections(zt)...)
	}

	if len(sections) == 0 {
		return nil
	}

	return &uci.Package{
		Name:     "zerotier",
		Sections: sections,
	}
}

func buildZerotierSection(zt *zerotierv1.ZerotierNetwork) *uci.Section {
	if zt == nil {
		return nil
	}

	name := sanitizeIdentifier(zt.GetName())
	if name == "" {
		return nil
	}

	section := uci.NewSection("zerotier", name)

	// enabled field (inverse of disabled) - TODO: check if zerotier has disabled field
	helpers.SetBoolValue(section, "enabled", true)

	helpers.SetString(section, "config_path", "/etc/openwisp/zerotier")
	helpers.SetBoolValue(section, "copy_config_path", true)

	// join list - network IDs from all networks
	// TODO: This needs to be implemented based on zerotier proto structure

	return section
}

func buildZerotierNetworkSections(zt *zerotierv1.ZerotierNetwork) []*uci.Section {
	// TODO: Implement network sections for ZeroTier > 1.14
	// Each network should be a separate section of type "network"
	return nil
}

// FromAST 根据 UCI 文档重建领域模型。
func FromAST(doc *uci.Document) (*Config, error) {
	if doc == nil {
		return nil, nxerrors.New(nxerrors.KindParse, fmt.Errorf("document is nil"))
	}
	return nil, nxerrors.ErrNotImplemented
}

// ToProto 输出 NetJSON proto。
func (c *Config) ToProto() (*openwrtv1.OpenWrtConfig, error) {
	if c == nil {
		return nil, nxerrors.New(nxerrors.KindInternal, fmt.Errorf("config is nil"))
	}
	return nil, nxerrors.ErrNotImplemented
}
