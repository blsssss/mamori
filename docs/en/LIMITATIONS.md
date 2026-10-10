[Русский](../ru/LIMITATIONS.md) | English

# Limitations

What the checks do not prove and where they may be wrong, with the reasons. How the checks work:
[CHECKS.md](CHECKS.md).

## internet

1. **A TUN or proxy client is recognized only by TTL 255.** Only an ICMP reply with TTL 255
   counts as local. A TUN that does not answer ping, or answers with another TTL, goes unnoticed,
   and a TCP connection through it counts as reaching a remote host. Conversely, a device on the
   same link that answers for a public address with TTL 255 counts as local.
   Reason: TCP does not show the TTL to the application, only ICMP gives a sign of a local answer.
2. **The Windows system proxy is not used.** The HTTP probe takes a proxy only from the
   environment variables `HTTP_PROXY` and `NO_PROXY` (the probe URL is plain HTTP). Where the
   internet is reachable only through a proxy set in the Windows settings, the probe fails and the
   verdict is `offline`, `dns_broken` or `limited`, depending on which direct probes still pass.
   Reason: `http.ProxyFromEnvironment` of the Go standard library.
3. **IPv4 only for adapters, ICMP and TCP.** On an IPv6-only network only the HTTP probe can show
   a connection. Reason: the adapter list is requested for `AF_INET`, the ICMP and TCP targets are
   IPv4 addresses.
4. **Dependence on the Microsoft NCSI servers.** If `www.msftconnecttest.com` is blocked or
   unavailable, the verdict is `limited`, `captive` or, when the name is blocked at the DNS
   level, `dns_broken`, although the internet works. Reason: only the exact answer of that server
   gives `online`, and the DNS probe resolves the same name.
5. **`net.dns.ok` may come from a cache.** The name is resolved by the system resolver, which
   answers from its cache. A cached answer can hide a broken DNS server: when a host is reachable
   and the HTTP probe fails, the verdict is `limited` instead of `dns_broken`; when the HTTP probe
   also gets the address from the cache and passes, the verdict is `online`. Without a reachable
   host the verdict is `offline` either way. Reason: the check does not query a DNS server
   directly.

## inventory

1. **`productState` is decoded by a heuristic.** Microsoft does not document the layout of the
   WMI field `productState`, decoding bits 12-15 and 4-7 is common community practice. With
   another layout the state may be wrong or `unknown`. Reason: the documented WSC API is not
   always available, WMI remains the fallback.
2. **Without Security Center the levels rest on Windows components.** If WSC fails for any
   reason, the lists come from WMI. If WMI fails as well (`inv.wsc.unavailable` and
   `inv.wmi.unavailable`), the AV level rests on Defender alone, a third-party AV is not counted,
   and without a protecting Defender the verdict is `inv.verdict.unknown`. A third-party FW is not
   visible either, the FW level rests on Windows Firewall, and without an enforcing Windows
   Firewall the verdict is `inv.verdict.unknown` as well. This is the expected case on Windows
   Server, which has no Security Center.
3. **Security Center shows what the products report about themselves.** A product may report
   `on` with a broken engine or keep a stale state: Defender stays in the list after its service
   stops (as `snoozed`). Defender entries are discounted while `WinDefend` is stopped, other
   products get no such correction. Reason: there is no other source for the state of
   third-party products, which is why `firewall` and `antivirus` test the actual work.
4. **Defender is judged by four signs.** The service, real-time protection, the antivirus flag
   and the running mode are used. Exclusions, cloud protection and tamper protection are not
   read. If the state of the `WinDefend` service cannot be queried, or the Defender WMI does not
   answer while the service runs, nothing is claimed about protection
   (`inv.defender.unavailable`).
5. **Product traces come from a fixed table.** Products outside the table of 19 entries are not
   found, a folder may remain after a product was uninstalled. Traces do not affect the verdict.
6. **A firewall not registered with Security Center is invisible.** The verdict counts only
   Windows Firewall and third-party firewalls from Security Center.

## firewall

1. **The rule test exercises only Windows Firewall.** A third-party firewall has its own rules.
   When Windows Firewall is not enforcing and a third-party firewall is on, the rule test is
   skipped (`fw.rule.third_party`), and only the policy test can prove the firewall works. When
   both are on, only Windows Firewall is tested.
2. **Group policy.** With `NET_FW_MODIFY_STATE_GP_OVERRIDE` local rules are not applied, and the
   test is skipped (`fw.rule.gp_override`). If that value cannot be read, the test runs, and an
   ignored rule looks like `not_enforced`.
3. **Outbound TCP only.** The rule and the policy test concern outbound TCP connections. Inbound
   filtering, UDP and ICMP are not tested, the default inbound action is only read.
4. **Application of a new rule is proved, not the quality of the policy.** The rule test shows
   that the firewall applies a rule to the program. The user's rules are not assessed, the
   policy test covers only the addresses entered.
5. **Coincidence with a network outage.** Any connection error while the rule is in place counts
   as a block. The restore after removing the rule rules out a dropped network, but an outage
   that starts right after the rule is added and ends before the next connection would be taken
   for a block.
6. **Policy test classification.** Only `WSAEACCES` and a timeout while the network works count
   as evidence. A firewall that answers with a refusal or a reset gives `inconclusive`. Blocking
   by DNS gives `inconclusive` too. A silent drop by the provider or a dead host while a firewall
   is on counts as a block. Connections go direct, without a proxy. Reason: any hop on the way
   can send a refusal or a reset, and a silent drop looks the same wherever it happens.
7. **At least one control host is needed.** The rule test and the timeout classification need
   one of `1.1.1.1:443`, `8.8.8.8:443`, `77.88.8.8:443`. When none is reachable, the result is
   `fw.rule.control_failed` and `inconclusive`.
8. **A rule may stay after a crash.** Cancelling the check or its 3 minute timeout only stops the
   window from waiting: the elevated copy is not killed and removes its rule itself. If the copy
   crashes or is killed from outside, the deferred removal does not run. The rule then stays
   until the next rule test, which removes it as its first step. It applies only to
   `mamori.exe` and one address.
9. **COM calls of the rule test have no timeout.** A hung `INetFwPolicy2` call holds the copy
   that made it. Through the elevated copy the window stops waiting on cancel; in an elevated
   process and in `--probe` the check waits for the call.
10. **Loopback is not tested.** All targets are public. Whether Windows Firewall filters
    `127.0.0.1` and `::1` traffic is not tested, and the program claims nothing about it.

## antivirus

1. **EICAR depends on on-access scanning and the wait time.** An AV without real-time file
   protection, with `%TEMP%` or `.txt` excluded, or one that scans files only on execution, gives
   `av.eicar.not_detected`. So does an AV that reacts slower than the wait (15 s by default,
   configurable).
2. **`ERROR_ACCESS_DENIED` is ambiguous.** Code 5 counts as a block by the antivirus because some
   AVs answer that way, but it can have other causes: folder permissions, another security tool.
   Any change of the file content counts as a reaction too.
3. **AMSI depends on a registered provider.** `AmsiScanString` hands the string to the AMSI
   provider. A third-party AV without an AMSI provider gives `av.amsi.not_detected` or
   `av.amsi.unavailable` although it protects, and the verdict is `partial`. Results in the
   `AMSI_RESULT_BLOCKED_BY_ADMIN` range (16384-20479) count as not detected. Without `amsi.dll` or
   its exports AMSI is unavailable.
4. **Test samples only.** Every antivirus knows EICAR, and the AMSI test string is recognised by
   the AMSI provider when one is registered (item 3). Detecting the samples
   shows that the scanning chain works, but says nothing about the detection quality for real
   threats.
5. **Notifications and logs.** The AV shows a threat notification, logs the event and may
   quarantine the file; in a managed environment the detection may reach the security team's
   console. Reason: for the AV this is an ordinary detection. The installer's finish page warns
   about the notification.
6. **The file may stay.** If the AV holds the folder for longer than four removal attempts, the
   folder `%TEMP%\mamori-av-*` stays (finding `av.eicar.cleanup_failed` with the path).
7. **A long wait is cut by the timeout.** The window gives one check 3 minutes, `--probe` gives
   the whole run 5 minutes. An EICAR wait longer than that ends the check with `common.cancelled`.

## Implementation and CI

1. **A hung COM or WMI call of `secprod` holds a goroutine.** This concerns the reads of Security
   Center, the Windows Firewall state and Defender, in `inventory` and in the state step of
   `firewall`. Waiting stops after 15 s, but the goroutine, and under `withCOM` of `secprod` its
   locked OS thread, stay until the call returns. The `wmi` library runs queries one at a time
   under a shared lock, so a hung query delays the following WMI queries, and each of them stops
   waiting after 15 s. The COM calls of the rule test have no timeout, see `firewall`, item 9.
2. **GitHub runners have Defender switched off.** Windows runners run Windows Server without
   Security Center, and Defender real-time protection is off there. That is why the unit tests
   in CI run with `-short`, and `e2e` checks the structure of the results, the cleanup of rules
   and the block under an enforcing Windows Firewall, but cannot show a working AV: the
   `antivirus` check is expected to fail there.
3. **Live tests do not change the system.** `go test` without `-short` reads Security Center,
   the services and the firewall settings, opens network connections and scans the AMSI string in
   memory, but adds no rules and writes no EICAR. The rule test sequence is tested against a fake firewall, the AV reaction to
   EICAR is not reproduced in automated tests.
