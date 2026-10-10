[Русский](../ru/CHECKS.md) | English

# Checks

Four checks and a summary. How the application is built: [ARCHITECTURE.md](ARCHITECTURE.md),
limits of the checks: [LIMITATIONS.md](LIMITATIONS.md).

| Check | Package | What it finds out | Administrator rights | What it changes |
|---|---|---|---|---|
| `internet` | `internal/netcheck` | whether the internet connection works, and if not, what is in the way | not needed | nothing |
| `inventory` | `internal/secprod` | which antivirus (AV) and firewall (FW) products are present and in what state | not needed | nothing, read only |
| `firewall` | `internal/fwprobe` | whether the firewall actually filters traffic | for the rule test only | a temporary Windows Firewall rule |
| `antivirus` | `internal/avprobe` | whether the antivirus reacts to test samples | not needed | a temporary file in `%TEMP%` |

Common codes:

| Code | Status | When |
|---|---|---|
| `common.cancelled` | `skip` | the check was cancelled or ran out of time |
| `common.unsupported_os` | `skip` | the program runs on something other than Windows |

Network connections of the checks:

| Address | Protocol | Check |
|---|---|---|
| `www.msftconnecttest.com` | DNS; HTTP, port 80, `/connecttest.txt` | `internet` |
| `dns.msftncsi.com` | DNS | `internet` |
| `1.1.1.1`, `8.8.8.8`, `77.88.8.8` (Cloudflare, Google, Yandex) | ICMP echo; TCP 443 | `internet`; `firewall` (control host) |
| addresses entered by the user | TCP, HTTP(S) | `firewall`, policy test |
| proxy from `HTTP_PROXY`, unless `NO_PROXY` excludes the host | HTTP, instead of the direct connection to `www.msftconnecttest.com` | `internet` |
| DNS servers of the system | DNS: the names above and the host names from the policy list, through the system resolver | `internet`, `firewall` |

The checks make no other network connections.

## internet: internet connection

### What and why

An adapter with an address does not mean there is internet. The strongest evidence is the
exact answer of the Microsoft NCSI test server (the service Windows itself uses to detect a
connection) to a plain HTTP request. It only arrives over a working path to the internet. A
captive portal answers that request itself: with a redirect, with its own page or with code 511.
The other probes tell the causes apart: no adapter, broken DNS, no route.

### Probes

All probes run in parallel, each with a 3 s timeout, so the whole check takes about one timeout.

| Probe | Source | Target |
|---|---|---|
| adapters | `GetAdaptersAddresses` (iphlpapi), IPv4 only | local interfaces |
| DNS | system resolver | `www.msftconnecttest.com` |
| NCSI DNS | system resolver, IPv4 | `dns.msftncsi.com`, expected `131.107.255.255` |
| ICMP | `IcmpSendEcho2` (iphlpapi.dll), no administrator rights, 32 bytes as `ping.exe` sends | `1.1.1.1`, `8.8.8.8`, `77.88.8.8` at once |
| TCP | TCP connection | `1.1.1.1:443`, `8.8.8.8:443`, `77.88.8.8:443` at once |
| HTTP | GET without following redirects, a fresh connection, proxy from environment variables | `http://www.msftconnecttest.com/connecttest.txt`, body `Microsoft Connect Test` |

Details:

- an adapter counts when it is up, not loopback, and has an IPv4 address outside
  `169.254.0.0/16` (Windows assigns itself such an address when DHCP did not answer);
- three hosts of different operators: one of them may be filtered or throttled on the way;
- an ICMP reply counts only when it comes from the pinged address with status `IP_SUCCESS`;
- a reply with TTL 255 crossed no router: this machine (the TUN adapter of a VPN or proxy client)
  or a device on the same link answered for a public address. Such a reply gives
  `net.icmp.local`, not `net.icmp.ok`;
- HTTP: code 200 with the body exactly `Microsoft Connect Test` gives `net.http.ok`; code 400 or
  higher, except 511, gives `net.http.fail`; anything else (redirect, another body, 204, 511)
  gives `net.http.captive`. The first 512 bytes of the body are read.

### Findings

| Code | Status | Params | When |
|---|---|---|---|
| `net.adapter.ok` | `pass` | `name`, `ipv4`, `gateway` | one per usable adapter |
| `net.adapter.none` | `fail` | | no usable adapter |
| `net.adapter.error` | `error` | `error` | the adapter list could not be read |
| `net.dns.ok` | `pass` | `host`, `addrs`, `ms` | the name resolved |
| `net.dns.fail` | `fail` | `host`, `error` | the name did not resolve |
| `net.ncsi_dns.ok` | `pass` | `host`, `addr` | the answers include `131.107.255.255` |
| `net.ncsi_dns.mismatch` | `warn` | `host`, `addr`, `want` | another address: DNS is rewritten on the way, as portals do |
| `net.ncsi_dns.fail` | `fail` | `host`, `error` | the name did not resolve |
| `net.icmp.ok` | `pass` | `target`, `rtt_ms`, `ttl` | first reply from a remote host |
| `net.icmp.local` | `warn` | `target`, `rtt_ms`, `ttl` | only replies with TTL 255 |
| `net.icmp.fail` | `warn` | `target`, `error` | no reply; ICMP is often filtered on purpose, hence not `fail` |
| `net.tcp.ok` | `pass` | `target`, `ms` | first established connection |
| `net.tcp.fail` | `fail` | `target`, `error` | no connection was established |
| `net.http.ok` | `pass` | `url`, `ms` | the exact NCSI answer |
| `net.http.captive` | `warn` | `url`, `status`, `location` | an answer that looks like a captive portal |
| `net.http.fail` | `fail` | `url`, `error` | network error or code 400 or higher, except 511 |

### Verdict

Function `netcheck.verdict`. A host counts as reached when there is `net.icmp.ok`, or there is
`net.tcp.ok` and no `net.icmp.local`: a TUN that answers ping for public addresses accepts TCP
connections itself as well.

| Order | Condition | Status | Code |
|---|---|---|---|
| 1 | `net.http.ok` present | `pass` | `net.verdict.online` |
| 2 | `net.http.captive` present | `warn` | `net.verdict.captive` |
| 3 | `net.adapter.none` present or no host reached | `fail` | `net.verdict.offline` |
| 4 | no `net.dns.ok` | `warn` | `net.verdict.dns_broken` |
| 5 | otherwise | `warn` | `net.verdict.limited` |

Examples from the table tests:

| Findings | Verdict |
|---|---|
| ICMP fail, TCP ok, HTTP ok | `online` |
| no adapter, DNS fail, TCP fail, HTTP ok | `online`: HTTP outweighs the rest |
| ICMP local, TCP ok, HTTP ok | `online` |
| NCSI DNS mismatch, TCP ok, HTTP captive | `captive` |
| DNS fail, ICMP ok, TCP ok, HTTP fail | `dns_broken` |
| DNS ok, ICMP fail, TCP ok, HTTP fail | `limited` |
| adapter list error, DNS ok, TCP ok, HTTP fail | `limited` |
| no adapter, DNS ok, ICMP ok, TCP ok, HTTP fail | `offline` |
| ICMP local, TCP ok, HTTP fail | `offline`: the TUN answers itself |
| DNS ok, ICMP fail, TCP fail, HTTP fail | `offline`: DNS answered from the local cache |

## inventory: AV and FW presence

### What and why

The check finds out which AV and FW products Windows considers installed and on, and the state
of Windows Firewall and Microsoft Defender. This is evidence of registration and state, not of
work: the `firewall` and `antivirus` checks prove work. The check changes nothing and needs no
administrator rights: services are opened with minimal rights (`SC_MANAGER_CONNECT` and
`SERVICE_QUERY_STATUS`, `SC_MANAGER_ENUMERATE_SERVICE`).

### Sources

| Source | API | What is read |
|---|---|---|
| Security Center, main | COM `IWSCProductList` (wscapi), CLSID `{17072F7B-9ABE-4A74-A261-1EB76B55107A}`, IID `{722A338C-6E8E-4E72-AC27-1417FB0C81C2}`; `Initialize` with `WSC_SECURITY_PROVIDER_ANTIVIRUS` (0x4) and `WSC_SECURITY_PROVIDER_FIREWALL` (0x1), `get_Count`, `get_Item`; on `IWscProduct`: `get_ProductName`, `get_ProductState`, `get_SignatureStatus`, `get_RemediationPath`, `get_ProductStateTimestamp` | AV and FW products registered with Windows |
| Security Center, fallback | WMI `root\SecurityCenter2`, classes `AntiVirusProduct`, `FirewallProduct`: `displayName`, `productState`, `pathToSignedProductExe`, `timestamp` | the same; `productState` decoded by a heuristic |
| Windows Firewall | service `mpssvc` (`QueryServiceStatus`); COM `HNetCfg.FwPolicy2` (`INetFwPolicy2`): `CurrentProfileTypes`, `FirewallEnabled`, `DefaultInboundAction`, `DefaultOutboundAction` for the domain (0x1), private (0x2) and public (0x4) profiles | effective profile settings |
| Microsoft Defender | service `WinDefend`; WMI `root\Microsoft\Windows\Defender`, class `MSFT_MpComputerStatus`: `AMRunningMode`, `AntivirusEnabled`, `RealTimeProtectionEnabled`, `AntivirusSignatureAge`, `AntivirusSignatureVersion` | Defender state apart from Security Center |
| Traces on disk | folders (`registry.ExpandString`, `os.Stat`), processes (`CreateToolhelp32Snapshot`), services (`EnumServicesStatusEx`) | table of known products |

Reading rules:

- both lists (AV and FW) come from one source: when WSC fails for either kind, both come from
  WMI. A report never mixes the sources;
- Windows Firewall is dropped from the Security Center lists: it is read from its own service
  and policy, and in the list it would pass for a third-party FW. WSC gives it a localized name
  ("Брандмауэр Windows" on Russian Windows), so it is recognised by its remediation path
  `%windir%\system32\firewall.cpl`; for WMI, which has no path, by the exact names
  `Windows Firewall`, `Windows Defender Firewall`, `Microsoft Defender Firewall` and their
  Russian forms, case-insensitive;
- profiles are read through `INetFwPolicy2`, not the WMI class `MSFT_NetFirewallProfile`: the
  WMI class reports "not configured" for defaults, `INetFwPolicy2` returns the effective values;
- when the `mpssvc` service is not running, a failure to read the profiles is not an error:
  without its service the firewall filters nothing;
- Windows Firewall is enforcing (`Enforcing`) when `mpssvc` runs, at least one profile is active
  and every active profile is enabled;
- Defender is protecting when `WinDefend` runs, real-time protection and the antivirus are on,
  and it is not in passive mode. Passive mode: `AMRunningMode` contains `passive`
  (`Passive Mode`, `SxS Passive Mode`) or `EDR Block`.

### Product state

WSC returns the enumerations from `iwscapi.h`:

| Value | State (`WSC_SECURITY_PRODUCT_STATE`) | Signatures (`WSC_SECURITY_SIGNATURE_STATUS`) |
|---|---|---|
| 0 | `on` | `out_of_date` |
| 1 | `off` | `up_to_date` |
| 2 | `snoozed` | `unknown` |
| 3 | `expired` | `unknown` |
| other | `unknown` | `unknown` |

When WSC does not return the signature status, it is `unknown`. In WMI, `productState` is
decoded by a heuristic: Microsoft does not document the layout of the field.

| Bits | Value | Result |
|---|---|---|
| 12-15 (`& 0xF000`) | 0x0, 0x1, 0x2, 0x3, other | `off`, `on`, `snoozed`, `expired`, `unknown` |
| 4-7 (`& 0x00F0`) | 0x0, 0x1, other | `up_to_date`, `out_of_date`, `unknown` |

Examples: `0x061100` (397568) Defender `on`; `0x062100` (401664) `snoozed`, which is how
Defender looks with its service stopped; `0x061110` `on`, signatures out of date; `0x041000` a
third-party product `on`.

### Findings

| Code | Status | Params |
|---|---|---|
| `inv.wsc.unavailable` | `warn` | `error`: WSC failed, the lists come from WMI |
| `inv.wmi.unavailable` | `warn` | `error`: WMI failed too, no product lists |
| `inv.av.product` | see the table below | `name`, `state`, `signature`, `source` (`wsc` or `wmi`), `path`, `timestamp` |
| `inv.fw.product` | see the table below | `name`, `state`, `source` |
| `inv.winfw.service` | `pass` when `running`, otherwise `fail` | `state`: `running`, `stopped`, `start_pending`, `stop_pending`, `continue_pending`, `pause_pending`, `paused`, `unknown`, `missing` |
| `inv.winfw.unavailable` | `warn` | `error` |
| `inv.winfw.profile` | `pass` enabled; `fail` disabled and active; `warn` disabled and inactive | `profile`, `enabled`, `active`, `inbound`, `outbound` (`block`, `allow`, `unknown`) |
| `inv.defender.status` | see the table below | `realtime`, `antivirus`, `service`, `mode`, `signature_age` (days), `signature_version` |
| `inv.defender.absent` | `skip` | `error`: no `WinDefend` service or no WMI namespace (`WBEM_E_INVALID_NAMESPACE`, 0x8004100E) |
| `inv.defender.unavailable` | `warn` | `service` (`unknown` when the service state could not be queried), `error`: the `WinDefend` service could not be queried, or it runs but `MSFT_MpComputerStatus` could not be read; nothing is claimed about protection |
| `inv.files.found` | `pass` | `vendor`, `kind` (`av`, `fw`, `suite`), `evidence` (`path`, `process`, `service`), `value` |
| `inv.files.none` | `warn` | no traces of known products |
| `inv.files.unavailable` | `warn` | `evidence` (`process` or `service`), `error` |

Status of a Security Center product:

| State | AV | FW |
|---|---|---|
| `on`, signatures `up_to_date` | `pass` | `pass` |
| `on`, signatures `out_of_date` or `unknown` | `warn` | `pass`: a firewall has no signatures |
| `snoozed`, `unknown` | `warn` | `warn` |
| `off`, `expired` | `fail` | `fail` |

Defender status (first matching condition):

| Condition | Status |
|---|---|
| passive mode | `warn` |
| protecting | `pass` |
| not protecting, but another AV (not Defender) is on | `warn`: Defender steps aside for another AV |
| otherwise | `fail` |

When `WinDefend` is not running and the Defender WMI does not answer, the state is taken from
the service alone: a stopped Defender protects nothing.

### Product traces

`inv.files.found` findings are informational and do not affect the verdict. Paths start with
the variables `%ProgramFiles%`, `%ProgramFiles(x86)%`, `%ProgramData%`, process names are
compared case-insensitively, service names are matched against lowercase `path.Match`
patterns. Kind `suite` means the product line has a firewall in some editions.

| Product | Kind | Folders | Processes | Services |
|---|---|---|---|---|
| Kaspersky | suite | `%ProgramFiles(x86)%\Kaspersky Lab`, `%ProgramFiles%\Kaspersky Lab`, `%ProgramData%\Kaspersky Lab` | avp.exe, avpui.exe, kavfs.exe | avp*, kavfs |
| Dr.Web | suite | `%ProgramFiles%\DrWeb`, `%ProgramFiles(x86)%\DrWeb`, `%ProgramData%\Doctor Web` | dwservice.exe, dwengine.exe, spideragent.exe | drweb* |
| ESET | suite | `%ProgramFiles%\ESET`, `%ProgramData%\ESET` | ekrn.exe, egui.exe | ekrn |
| Avast | suite | `%ProgramFiles%\Avast Software`, `%ProgramData%\Avast Software` | avastsvc.exe, avastui.exe | avast*, aswbidsagent |
| AVG | suite | `%ProgramFiles%\AVG`, `%ProgramData%\AVG` | avgsvc.exe, avgui.exe | avg* |
| Avira | av | `%ProgramFiles%\Avira`, `%ProgramFiles(x86)%\Avira`, `%ProgramData%\Avira` | avira.servicehost.exe, avira.systray.exe, avguard.exe | avira*, antivir* |
| Bitdefender | suite | `%ProgramFiles%\Bitdefender`, `%ProgramData%\Bitdefender` | bdagent.exe, vsserv.exe, bdservicehost.exe | vsserv |
| Norton | suite | `%ProgramFiles%\Norton`, `%ProgramFiles%\Norton Security`, `%ProgramFiles(x86)%\Norton Security` | nortonsvc.exe, nortonui.exe, nortonsecurity.exe | norton*, nswscsvc |
| McAfee | suite | `%ProgramFiles%\McAfee`, `%ProgramFiles(x86)%\McAfee`, `%ProgramData%\McAfee` | mcshield.exe, mfemms.exe, mfevtps.exe | mcafee*, mcshield, mfemms, mfevtp, mfefire |
| Comodo | suite | `%ProgramFiles%\COMODO`, `%ProgramFiles(x86)%\COMODO` | cmdagent.exe, cis.exe | cmdagent |
| 360 Total Security | av | `%ProgramFiles(x86)%\360\Total Security`, `%ProgramFiles%\360\Total Security` | qhactivedefense.exe, qhsafetray.exe, 360tray.exe | qhactivedefense |
| Sophos | av | `%ProgramFiles%\Sophos`, `%ProgramFiles(x86)%\Sophos`, `%ProgramData%\Sophos` | sophoshealth.exe, sspservice.exe, savservice.exe | sophos*, savservice, sspservice |
| Malwarebytes | av | `%ProgramFiles%\Malwarebytes\Anti-Malware`, `%ProgramData%\Malwarebytes\MBAMService` | mbamservice.exe, mbamtray.exe, malwarebytes.exe | mbamservice |
| Microsoft Defender | av | `%ProgramFiles%\Windows Defender`, `%ProgramData%\Microsoft\Windows Defender` | msmpeng.exe, nissrv.exe | windefend, wdnissvc |
| Windows Firewall Control | fw | `%ProgramFiles%\Malwarebytes\Windows Firewall Control` | wfc.exe | _wfcs |
| ZoneAlarm | fw | `%ProgramFiles(x86)%\CheckPoint\ZoneAlarm` | zatray.exe, vsmon.exe | vsmon |
| TinyWall | fw | `%ProgramFiles(x86)%\TinyWall`, `%ProgramFiles%\TinyWall` | tinywall.exe | tinywall |
| simplewall | fw | `%ProgramFiles%\simplewall` | simplewall.exe | |
| GlassWire | fw | `%ProgramFiles(x86)%\GlassWire`, `%ProgramFiles%\GlassWire` | glasswire.exe, gwctlsrv.exe | glasswire* |

### Verdict

Function `secprod.verdict` first finds an AV level and a FW level: `ok`, `degraded`, `unknown`,
`none`.

AV level:

| Condition | Level |
|---|---|
| Security Center unread, Defender protecting | `ok` |
| Security Center unread, otherwise | `unknown` |
| an AV `on` with signatures `up_to_date` | `ok` |
| an AV `on` with other signatures, or `snoozed` | `degraded` |
| an AV `unknown` | `unknown` |
| no AV, or all `off`, `expired` | `none` |

Defender entries in the Security Center list are skipped when the `WinDefend` service is known
and not running: Security Center keeps listing Defender after its service stops. Precedence:
`ok`, then `degraded`, then `unknown`.

FW level:

| Condition | Level |
|---|---|
| Windows Firewall enforcing | `ok` |
| a third-party FW `on` | `ok` |
| a third-party FW `snoozed` | `degraded` |
| a third-party FW `unknown` | `unknown` |
| none of these, but Windows Firewall or Security Center unread | `unknown` |
| otherwise | `none` |

| Order | Condition | Status | Code |
|---|---|---|---|
| 1 | AV `none` and FW `none` | `fail` | `inv.verdict.none` |
| 2 | AV `none` | `fail` | `inv.verdict.no_av` |
| 3 | FW `none` | `fail` | `inv.verdict.no_fw` |
| 4 | AV or FW `unknown` | `error` | `inv.verdict.unknown` |
| 5 | AV or FW `degraded` | `warn` | `inv.verdict.outdated` |
| 6 | otherwise | `pass` | `inv.verdict.ok` |

Examples from the table tests:

| Situation | Verdict |
|---|---|
| AV `on`, Windows Firewall enforcing | `ok` |
| AV `on`, third-party FW `on`, firewall service stopped | `ok` |
| Defender `off` and another AV `on` | `ok` |
| AV signatures out of date | `outdated` |
| AV `unknown` next to an AV `snoozed` | `outdated` |
| only a disabled Defender | `no_av` |
| Defender `on` in Security Center, `WinDefend` stopped | `no_av` |
| Defender stopped, another AV `on` | `ok` |
| active firewall profile disabled | `no_fw` |
| firewall service stopped, third-party FW `unknown` | `unknown` |
| Security Center unread, Defender protecting | `ok` |
| Security Center unread, Defender absent or unread | `unknown` |
| Windows Firewall unread, third-party FW `on` | `ok` |

## firewall: FW at work

### What and why

A firewall that is on does not necessarily filter traffic. The check has three steps:

1. state: Windows Firewall and third-party firewalls from Security Center;
2. policy test: try to reach resources the policy must forbid (the user gives the list);
3. rule test: a temporary rule blocks the program's own connection to a control host. The
   connection works before the rule, fails with the rule, and works again once the rule is
   removed. The change is caused by the rule, so the firewall applies its rules.

### Step 1. State

Same sources as `inventory`: the `mpssvc` service and `INetFwPolicy2`, third-party firewalls
from WSC, and from WMI `root\SecurityCenter2` when WSC fails.

| Code | Status | Params | When |
|---|---|---|---|
| `fw.state.enforcing` | `pass` | `profiles` | Windows Firewall enforcing, the active profiles listed |
| `fw.state.off` | `fail` | `profile` | an active profile is disabled, one per profile |
| `fw.state.service` | `fail` | `state` | the `mpssvc` service is not running or missing (`missing`) |
| `fw.state.unavailable` | `warn` | `error` | state not read: `windows firewall: ...`, `windows firewall: no active profile` or `security center: ...` |
| `fw.state.third_party` | `pass` for `on`, `warn` for `unknown`, otherwise `fail` | `name`, `state` | a third-party FW from Security Center |

### Step 2. Policy test

The addresses come from the interface field `policyTargets` or the `-policy` flag. All
addresses are tried in parallel, with a 5 s timeout each.

| Entry | TCP connection | HTTP |
|---|---|---|
| `host` | `host:443` | no |
| `host:port` | `host:port` | no |
| IPv6, also in `[]` | `[address]:443` | no |
| `http://host/path` | `host:80` or the given port | GET of the entry |
| `https://host/path` | `host:443` or the given port | GET of the entry |

Every connection is fresh (`net.Dialer`). For a URL, the connection is followed by a GET without
a proxy, without following redirects and without connection reuse: the answer has to come from
the address itself.

Classification (function `policyFinding`):

| Outcome | Finding |
|---|---|
| connection established, for a URL any HTTP answer received | `fw.policy.reachable` (`fail`) `{target, ms}` |
| connection refused locally, `WSAEACCES` (10013) | `fw.policy.blocked` (`pass`) `{target, error, reason: wsaeacces}` |
| connection timeout (not DNS), a control host reachable | `fw.policy.blocked` (`pass`) `{target, error, reason: timeout}` |
| timeout, control hosts unreachable | `fw.policy.inconclusive` (`warn`) `{target, error}` |
| refused, reset, unreachable, DNS error, cancelled, invalid entry | `fw.policy.inconclusive` (`warn`) |
| HTTP error after an established connection | `fw.policy.inconclusive` (`warn`) |
| empty list | `fw.policy.none` (`skip`) |

Why:

- `WSAEACCES` comes from the local Windows filtering platform, it is evidence on its own;
- a refusal or a reset may come from the remote side or any hop on the way;
- after an established TCP connection the firewall has already let the flow out, so a later
  HTTP error proves nothing;
- a timeout proves filtering only while the network works. This is checked with one control
  round per run: TCP to `1.1.1.1:443`, `8.8.8.8:443`, `77.88.8.8:443` in parallel, 4 s, and the
  round runs only on the first timeout.

### Step 3. Rule test

Whether and where it runs:

| Condition | Action |
|---|---|
| Windows Firewall not enforcing or unread, and a third-party FW `on` | `fw.rule.third_party` (`warn`) `{name}`, the test is not run: third-party rules are not Windows Firewall rules |
| the process is already elevated | test in this process |
| window, `allowElevation` | elevated copy through UAC ([ARCHITECTURE.md](ARCHITECTURE.md#elevation)) |
| otherwise | `fw.rule.needs_admin` (`skip`) |

Sequence (`fwprobe.ruleTest`), COM objects `HNetCfg.FwPolicy2` and `HNetCfg.FWRule`:

1. Rules with the prefix `mamori-probe-` left by interrupted runs are removed; if there were
   any, the finding `fw.rule.cleanup {count}`.
2. If `LocalPolicyModifyState` equals `NET_FW_MODIFY_STATE_GP_OVERRIDE` (1), group policy does
   not apply local rules: `fw.rule.gp_override` (`skip`), the test stops. If the value cannot be
   read, the test goes on.
3. Control host: TCP to `1.1.1.1:443`, `8.8.8.8:443`, `77.88.8.8:443` in parallel, 4 s, the
   first one to answer is chosen. None answered: `fw.rule.control_failed` (`skip`).
4. A rule is added (`Rules.Add`):

   | Property | Value |
   |---|---|
   | `Name` | `mamori-probe-` and 8 random hex digits |
   | `Description` | `Temporary rule of the mamori firewall test, removed right after the test` |
   | `Grouping` | `mamori` |
   | `ApplicationName` | path of the running `mamori.exe` |
   | `Protocol` | 6 (TCP) |
   | `RemoteAddresses` | IP of the control host |
   | `RemotePorts` | port of the control host (443) |
   | `Direction` | 2 (outbound) |
   | `Action` | 0 (block) |
   | `Profiles` | 0x7FFFFFFF (all) |
   | `Enabled` | true |

5. Pause 300 ms, a fresh connection to the control host within 4 s: an error gives
   `fw.rule.blocked`, success gives `fw.rule.not_enforced`.
6. The rule is removed: `fw.rule.removed` or `fw.rule.remove_failed`.
7. Only after a block and a successful removal: pause 300 ms, a fresh connection: success gives
   `fw.rule.restored`, an error gives `fw.rule.not_restored`.

Every connection after the rule is added is fresh: a connection opened earlier would bypass the
rule. Without step 7 a chance network drop would look like a block.

| Code | Status | Params |
|---|---|---|
| `fw.rule.cleanup` | `pass` | `count` |
| `fw.rule.gp_override` | `skip` | |
| `fw.rule.control_failed` | `skip` | `target`, `error` |
| `fw.rule.added` | `pass` | `name`, `target` |
| `fw.rule.add_failed` | `error` | `error`; also when `os.Executable`, COM initialization or opening the policy fails |
| `fw.rule.blocked` | `pass` | `target`, `error`, `ms` |
| `fw.rule.not_enforced` | `fail` | `target` |
| `fw.rule.removed` | `pass` | `name` |
| `fw.rule.remove_failed` | `error` | `name` (`mamori-probe-*` when the list could not be read), `error` |
| `fw.rule.restored` | `pass` | `target` |
| `fw.rule.not_restored` | `warn` | `target`, `error` |
| `fw.rule.third_party` | `warn` | `name` |
| `fw.rule.needs_admin` | `skip` | |
| `fw.rule.uac_declined` | `skip` | |
| `fw.rule.elevate_error` | `error` | `error` |

### Verdict

Function `fwprobe.verdict`. Terms:

- FW on: a `fw.state.*` finding with status `pass`;
- FW off: a `fw.state.*` finding with status `fail`;
- rule proved: `fw.rule.blocked` and `fw.rule.restored`;
- policy proved: a `fw.policy.blocked` with `reason: wsaeacces`, or with any reason while the FW
  is on (with no FW on, a silent drop by the provider or a dead host looks the same).

| Order | Condition | Status | Code |
|---|---|---|---|
| 1 | `fw.rule.not_enforced` or `fw.policy.reachable` present | `fail` | `fw.verdict.not_enforced` |
| 2 | rule proved or policy proved | `pass` | `fw.verdict.works` |
| 3 | FW off and not on | `fail` | `fw.verdict.disabled` |
| 4 | otherwise | `warn` | `fw.verdict.unverified` |

The elevated copy (`--probe firewall-rule`) decides its own verdict separately (`ruleVerdict`):
`fw.rule.not_enforced` gives `fail`, a proved rule `pass`, the rest `skip` with code
`fw.verdict.unverified`. In the window only its findings are used, the parent decides the
verdict.

Examples from the table tests:

| Situation | Verdict |
|---|---|
| enforcing, rule: block and restore | `works` |
| enforcing, block without restore | `unverified` |
| enforcing, rule not applied | `not_enforced` |
| rule works, but a policy address is reachable | `not_enforced` |
| enforcing, policy address dropped by timeout, UAC declined | `works` |
| profile off, no rights | `disabled` |
| profile off, policy address dropped by timeout | `disabled` |
| profile off, policy address refused locally (`WSAEACCES`) | `works` |
| third-party FW only, rule test skipped | `unverified` |
| third-party FW `on`, policy address dropped by timeout | `works` |
| state unread, policy address dropped by timeout | `unverified` |
| state unread, rule proved | `works` |

### Safety measures

- The rule applies only to `mamori.exe` and only to outbound TCP to one IP and port: the traffic
  of other programs is not affected.
- A unique name with the `mamori-probe-` prefix and a random part, group `mamori`, a description.
- The rule is removed right after the test, also on cancellation (`defer`). Rules of interrupted
  runs are removed at the start of every rule test. `scripts/e2e.ps1` checks that no rules are
  left.
- Administrator rights are needed by this step only and are requested through UAC on an
  explicit choice (`allowElevation`). Steps 1 and 2 only read the state and open connections.

## antivirus: AV at work

### What and why

- EICAR: the standard test file that antiviruses detect as malware by agreement, though it is
  harmless. When the AV removes, blocks or changes the file after it is written and opened,
  real-time file protection works.
- AMSI (Antimalware Scan Interface): the interface through which Windows and programs hand
  content to the registered antivirus. Detection of the AMSI test string shows that scanning
  through this interface works. The scan happens in memory, nothing is written to disk.

These are two different paths, so both are tested.

### EICAR

1. A folder `%TEMP%\mamori-av-<random>` is created and `eicar-test.txt` is written into it (mode
   `0600`). The EICAR string (68 bytes) is stored encoded and assembled in memory before the
   write, the buffer is zeroed afterwards.
2. A write error with Win32 code 5, 225 or 226 gives `av.eicar.blocked_on_write`. Any other error
   (or a failure to create the folder) gives `av.eicar.write_failed`.
3. Every 250 ms the file is opened and read again: the open triggers on-access scanning. Polling
   lasts up to `EicarWait` (15 s by default, field `eicarWaitSec`, flag `-eicar-wait`).

   | Read outcome | Action |
   |---|---|
   | the file is gone | `av.eicar.removed` |
   | error 5, 225, 226 | `av.eicar.blocked_on_read` |
   | content differs from EICAR (SHA-256 comparison) | `av.eicar.altered` |
   | content unchanged | keep waiting |
   | another error, e.g. the scanner holds the file | keep waiting |
   | time is up | `av.eicar.not_detected` |

4. The folder is removed, with up to three retries 250 ms apart: the scanner may still hold the
   file.

| Win32 code | Name |
|---|---|
| 5 | `ERROR_ACCESS_DENIED` |
| 225 | `ERROR_VIRUS_INFECTED` |
| 226 | `ERROR_VIRUS_DELETED` |

### AMSI

1. The exports `AmsiInitialize`, `AmsiOpenSession`, `AmsiScanString`, `AmsiCloseSession`,
   `AmsiUninitialize` are looked up in `amsi.dll`; if one is missing, AMSI is unavailable.
2. `AmsiInitialize("mamori")`, `AmsiOpenSession`, `AmsiScanString` with the AMSI test string from
   the Microsoft documentation (54 bytes, stored encoded), then the session is closed.
3. A result of `32768` (`AMSI_RESULT_DETECTED`) or more: detected. Less: not detected. A
   failing HRESULT: AMSI unavailable.

AMSI runs after EICAR.

### Findings

| Code | Status | Params |
|---|---|---|
| `av.eicar.blocked_on_write` | `pass` | `error` |
| `av.eicar.removed` | `pass` | `seconds` |
| `av.eicar.blocked_on_read` | `pass` | `error`, `seconds` |
| `av.eicar.altered` | `pass` | `seconds` |
| `av.eicar.not_detected` | `fail` | `seconds`, `error` (when the last read failed) |
| `av.eicar.write_failed` | `error` | `error` |
| `av.eicar.cleanup_failed` | `warn` | `path`, `error` |
| `av.amsi.detected` | `pass` | `result` |
| `av.amsi.not_detected` | `fail` | `result` |
| `av.amsi.unavailable` | `warn` | `error` |

`seconds` has one digit after the point.

### Verdict

Function `avprobe.verdict`, all nine combinations are covered by a table test.

| EICAR | AMSI | Status | Code |
|---|---|---|---|
| not run (folder or file could not be created) | any | `error` | `av.verdict.error` |
| detected | detected | `pass` | `av.verdict.works` |
| detected | not detected or unavailable | `warn` | `av.verdict.partial` |
| not detected | detected | `warn` | `av.verdict.partial` |
| not detected | not detected or unavailable | `fail` | `av.verdict.not_working` |

### Safety measures

- The EICAR and AMSI strings exist in the source and in the `.exe` only in encoded form (XOR with
  a key, then base64) and are assembled in memory right before use. The test compares SHA-256
  hashes only.
- The EICAR file is written only to its own temporary folder and removed with it. It is a text
  file (`.txt`) and is never run.
- The AMSI test string is never written to disk.

## Summary (`internal/report`)

`report.Summarize` builds a `Summary`: `version`, `commit`, `system`, `generated`, `status`,
`code`, `results`.

Rules of `report.verdict`:

- results with status `skip` are ignored;
- a `fail` of the `internet` check counts as `warn`: a missing connection alone does not make the
  machine unprotected, and a working connection does not make up for a broken firewall;
- the outcome is the most severe status of the rest (`pass` < `skip` < `warn` < `fail` < `error`).

| Condition | Status | Code |
|---|---|---|
| no check ran | `skip` | `report.none` |
| all `pass`, but some skipped | `warn` | `report.partial` |
| all `pass` | `pass` | `report.protected` |
| worst is `warn` | `warn` | `report.degraded` |
| worst is `error` | `error` | `report.error` |
| worst is `fail` | `fail` | `report.unprotected` |

`error` ranks above `fail`: if one check failed and another ended in an error, the summary is
`report.error`.

Examples from the table tests:

| Results | Summary |
|---|---|
| all four `pass` | `protected` |
| `internet` `pass`, `firewall` `skip` | `partial` |
| `internet` `fail`, `firewall` and `antivirus` `pass` | `degraded` |
| `firewall` `fail`, the rest `pass` | `unprotected` |
| `antivirus` `error`, `firewall` `fail` | `error` |
| `skip` only | `none` |
