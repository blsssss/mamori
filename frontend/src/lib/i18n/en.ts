import type { Dict } from './index'

export const en: Dict = {
  decimal: '.',

  status: {
    pass: 'passed',
    warn: 'warning',
    fail: 'failed',
    skip: 'skipped',
    error: 'error',
  },

  ui: {
    'app.name': 'mamori',

    // the same in every language: Japanese decoration and the classic VN quick menu
    'jp.mamori': '守り',
    'jp.reimu': '霊夢',
    'jp.start': '開始',
    'jp.log': '記録',
    'jp.config': '設定',
    'jp.quit': '終了',
    'jp.five': '五',
    'jp.locked': '未',
    'jp.exorcism': '妖怪退治',
    'jp.barrier': '博麗大結界',
    'jp.tea': '異常なし',
    'vn.auto': 'Auto',
    'vn.stop': 'Stop',
    'vn.log': 'Log',
    'vn.save': 'Save',
    'vn.config': 'Config',
    'vn.title': 'Title',
    'vn.quit': 'Quit',
    'lang.ru': 'Русский',
    'lang.en': 'English',
    'lang.ruShort': 'RU',
    'lang.enShort': 'EN',
    'cfg.policyPlaceholder': 'example.com\nhttps://example.org/\n203.0.113.10:8080',
    'app.unavailable': 'The Wails runtime is missing: this page is not running inside the mamori app.',

    'title.subtitle': 'PC protection check',
    'title.menu': 'Main menu',
    'title.start': 'Start',
    'title.log': 'Log',
    'title.config': 'Settings',
    'title.quit': 'Quit',
    'title.fan': 'Reimu Hakurei from Touhou Project © Team Shanghai Alice. Unofficial fan work.',
    'title.version': 'version {version}',

    'top.system': '{product}[[ {version}]][[, build {build}]][[, {arch}]]',
    'top.host': 'host {host}',
    'top.user': 'user {user}',
    'top.noInfo': 'system information unavailable',
    'top.admin': 'administrator',
    'top.adminHint': 'The program runs with administrator rights',
    'top.lang': 'Interface language',

    'check.internet': 'Internet connection',
    'check.inventory': 'Firewall and antivirus presence',
    'check.firewall': 'Firewall operation',
    'check.antivirus': 'Antivirus operation',
    'check.module': 'Module {n}',

    'chapters.label': 'Checks',
    'chapter.run': 'Run',
    'chapter.rerun': 'Run again',
    'chapter.running': 'Running…',
    'chapter.runAria': 'Run the check “{title}”',
    'chapter.rerunAria': 'Run again: the check “{title}”',
    'chapter.runningAria': 'Running: the check “{title}”',
    'chapter.selectAria': '{module}. {title}: {status}',
    'chapter.idle': 'not run',
    'chapter.active': 'running',
    'chapter.cg': 'Show the event “{caption}”',

    'overall.module': 'Module 5',
    'overall.label': 'Results output',
    'overall.pending': 'the verdict appears after the checks',
    'overall.open': 'Open the log',

    'box.label': 'Text window: what Reimu says about the selected check',
    'box.name': 'Reimu',
    'box.say': "Reimu's line",
    'box.counter': '{n} / {total}',
    'box.prev': 'Previous line',
    'box.next': 'Next line',
    'box.first': 'First line',
    'box.last': 'Last line',
    'box.raw': 'System message',
    'box.hint': 'Click, Enter or Space: next line. Arrows: back and forward, Home and End: first and last.',
    'box.waiting': 'The check is running',

    'say.intro.internet':
      'Module 1, the internet connection check. I will look at the network adapters, DNS, ping, a TCP connection and the NCSI HTTP probe, the same one Windows itself uses.',
    'say.intro.inventory':
      'Module 2, the firewall and antivirus presence check. I will read Windows Security Center, the Windows Firewall policy and the state of Microsoft Defender, and look for traces of known products in folders, processes and services.',
    'say.intro.firewall':
      'Module 3, the firewall operation check. I will try to reach resources the policy forbids and add a temporary rule that blocks a connection of this program, to see that the firewall enforces it.',
    'say.intro.antivirus':
      'Module 4, the antivirus operation check. I will write the harmless EICAR test file and pass the official AMSI test string to AMSI, then see how the antivirus reacts.',
    'say.hint':
      'To start, press “Run” next to this check on the left. AUTO below runs all four checks in order.',
    'say.notice.eicar':
      'I am about to write the harmless EICAR test file to a temporary folder. The antivirus may show a threat notification, that is expected: it is the test. I will delete the file afterwards.',
    'say.notice.uac':
      'The rule test needs administrator rights, so Windows will show a UAC prompt. If you decline it, the rule test is skipped and I check the rest.',
    'say.runError': 'The check could not run: the program returned an error.',

    'qm.label': 'Quick menu',
    'qm.auto': 'Run all four checks in order',
    'qm.stop': 'Stop the running check',
    'qm.log': 'Log: results output',
    'qm.save': 'Save the report to a file',
    'qm.config': 'Settings',
    'qm.title': 'Back to the title screen',
    'qm.quit': 'Quit the program',

    'log.title': 'Log',
    'log.subtitle': 'Module 5. Results output',
    'log.overall': 'Overall verdict',
    'log.generated': 'generated {date}',
    'log.checksRun': 'checks run: {n} of {total}',
    'log.loading': 'Summing up…',
    'log.notRun': 'The check has not run.',
    'log.running': 'The check is still running.',
    'log.elapsed': 'time {time}',
    'log.save': 'Save report',
    'log.copy': 'Copy',
    'log.clear': 'Clear results',
    'log.clearConfirm': 'Delete all check results?',
    'log.clearYes': 'Yes, clear',
    'log.clearNo': 'Cancel',
    'log.back': 'Back',
    'log.cgs': 'Events',
    'log.cgLocked': 'Event not unlocked yet',

    'save.title': 'Save report',
    'save.format': 'Report format',
    'save.txt': 'TXT, plain text',
    'save.html': 'HTML, for viewing and printing',
    'save.json': 'JSON, data for programs',
    'save.name': 'File name: {name}',
    'save.ok': 'Save',
    'save.cancel': 'Cancel',
    'save.done': 'Report saved: {path}',
    'save.failed': 'Could not save the report: {error}',
    'copy.done': 'Report copied to the clipboard.',
    'copy.failed': 'Could not copy the report.',
    'clear.done': 'Results cleared.',
    'summary.failed': 'Could not sum up the results: {error}',
    'run.busy': 'Wait until the running check ends.',

    'quit.title': 'Quit',
    'quit.running': 'A check is running. Stop it and quit the program?',
    'quit.ok': 'Quit',
    'quit.cancel': 'Stay',
    'quit.demo': 'The demo does not quit: this is a browser tab.',
    'quit.stopping':
      'Stopping the check: the program removes its temporary firewall rule and test file, then closes.',

    'cfg.title': 'Settings',
    'cfg.ui': 'Interface',
    'cfg.checks': 'Checks',
    'cfg.lang': 'Language',
    'cfg.speed': 'Text speed',
    'cfg.speedInstant': 'instant',
    'cfg.speedSlow': 'slow',
    'cfg.speedValue': '{n} of {max}',
    'cfg.speedSample': 'This is how I will type my lines in the text window.',
    'cfg.reduced': 'The system “reduce motion” setting is on: text appears at once and animations are off.',
    'cfg.cgs': 'Show illustrations after passed checks',
    'cfg.cgsHint':
      'A full-screen illustration for a couple of seconds when the firewall or antivirus check passes or the overall verdict is: protected. A click or any key closes it, the checks keep running.',
    'cfg.skipTitle': 'Skip the title screen on start',
    'cfg.policy': 'Resources the firewall policy forbids',
    'cfg.policyHint':
      'One host, host:port or URL per line, lines starting with # are ignored. List resources your policy is expected to block. The firewall check tries to connect to them: a successful connection means the policy does not work.',
    'cfg.policyCount': 'entries: {n}',
    'cfg.eicar': 'Wait for the EICAR reaction',
    'cfg.eicarValue': '{n} s',
    'cfg.eicarHint': 'How long to wait for the antivirus to delete, block or alter the test file.',
    'cfg.uac': 'Allow the UAC prompt for the rule test',
    'cfg.uacHint':
      'The rule test adds a temporary Windows Firewall rule that blocks only this program and removes it right after. That needs administrator rights: the program starts a copy of itself through a UAC prompt for the test only.',
    'cfg.reset': 'Reset settings',
    'cfg.back': 'Back',
    'cfg.storage': 'Storage is unavailable: the settings last until the program closes.',

    'cg.exorcism': 'The youkai is exorcised: the antivirus neutralised the EICAR test file.',
    'cg.barrier': 'The barrier holds: the firewall stopped a forbidden connection.',
    'cg.tea': 'Every ward is in place. Time for a cup of tea.',
    'cg.close': 'Click or press any key to close',
    'cg.aria': 'Event: {caption} Press to close.',

    'demo.ribbon': 'DEMO',
    'demo.note': 'demo data, scenario {scenario}',

    'time.seconds': '{n} s',
    'time.minutes': '{m} min {s} s',

    'rep.title': 'PC protection check report',
    'rep.program': 'Program',
    'rep.generated': 'Generated',
    'rep.system': 'Operating system',
    'rep.host': 'Computer',
    'rep.user': 'User',
    'rep.overall': 'Overall verdict',
    'rep.checksRun': 'Checks run',
    'rep.module': 'Module {n}',
    'rep.verdict': 'Verdict',
    'rep.elapsed': 'Duration',
    'rep.started': 'Started',
    'rep.findings': 'Findings',
    'rep.status': 'Status',
    'rep.text': 'Description',
    'rep.error': 'System message',
    'rep.code': 'Code',
    'rep.notRun': 'not run',
    'rep.demo': 'DEMO MODE: the data is made up, this is not the result of a real check.',
  },

  values: {
    state: {
      on: 'on',
      off: 'off',
      snoozed: 'snoozed',
      expired: 'expired',
      unknown: 'state unknown',
      running: 'running',
      stopped: 'stopped',
      start_pending: 'starting',
      stop_pending: 'stopping',
      continue_pending: 'resuming',
      pause_pending: 'pausing',
      paused: 'paused',
      missing: 'missing',
    },
    service: {
      running: 'running',
      stopped: 'stopped',
      start_pending: 'starting',
      stop_pending: 'stopping',
      continue_pending: 'resuming',
      pause_pending: 'pausing',
      paused: 'paused',
      missing: 'missing',
      unknown: 'in an unknown state',
    },
    signature: {
      up_to_date: 'signatures up to date',
      out_of_date: 'signatures out of date',
      unknown: 'signature state unknown',
    },
    source: {
      wsc: 'Windows Security Center, WSC API',
      wmi: 'Windows Security Center, WMI root\\SecurityCenter2',
    },
    profile: { domain: 'domain', private: 'private', public: 'public' },
    profiles: { domain: 'domain', private: 'private', public: 'public' },
    enabled: { true: 'on', false: 'off' },
    active: { true: 'active', false: 'inactive' },
    inbound: { block: 'block', allow: 'allow', unknown: 'unknown' },
    outbound: { block: 'block', allow: 'allow', unknown: 'unknown' },
    realtime: { true: 'on', false: 'off' },
    antivirus: { true: 'on', false: 'off' },
    mode: {
      Normal: 'normal mode',
      'Passive Mode': 'passive mode',
      'SxS Passive Mode': 'SxS passive mode',
      'EDR Block Mode': 'EDR block mode',
    },
    kind: {
      av: 'antivirus',
      fw: 'firewall',
      suite: 'antivirus, some editions with a firewall',
    },
    evidence: { path: 'folder', process: 'process', service: 'service' },
    'inv.files.unavailable:evidence': { process: 'the process list', service: 'the service list' },
    reason: {
      wsaeacces: 'the Windows filtering platform refused the connection (WSAEACCES, 10013)',
      timeout: 'the connection did not complete within 5 s while the control hosts were reachable',
    },
  },

  codes: {
    'common.unsupported_os': 'The check runs on Windows only.',
    'common.cancelled': 'The check was interrupted: the user stopped it or 3 minutes ran out.',

    'net.adapter.ok': 'Network adapter “{name}” is connected: IPv4 {ipv4}[[, gateway {gateway}]].',
    'net.adapter.none':
      'No connected network adapter with a working IPv4 address (169.254.x.x does not count).',
    'net.adapter.error': 'Could not list the network adapters (GetAdaptersAddresses).',
    'net.dns.ok': 'DNS: {host} resolves to {addrs} in {ms} ms.',
    'net.dns.fail': 'DNS: {host} does not resolve.',
    'net.ncsi_dns.ok': 'NCSI DNS probe: {host} points to the expected address {addr}.',
    'net.ncsi_dns.mismatch':
      'NCSI DNS probe: {host} points to {addr} instead of {want}, DNS answers are rewritten on the way.',
    'net.ncsi_dns.fail': 'NCSI DNS probe: {host} does not resolve.',
    'net.icmp.ok': 'ICMP: {target} answers ping in {rtt_ms} ms, TTL {ttl}.',
    'net.icmp.local':
      'ICMP: the reply from {target} came with TTL {ttl} in {rtt_ms} ms, so this computer (the TUN adapter of a VPN or proxy) or a device on the same link answered, not the remote host.',
    'net.icmp.fail':
      'ICMP: no ping reply from {target}. ICMP is often filtered on purpose, this alone does not mean there is no connection.',
    'net.tcp.ok': 'TCP: connected to {target} in {ms} ms.',
    'net.tcp.fail': 'TCP: could not connect to any of {target}.',
    'net.http.ok': 'NCSI HTTP probe: {url} returned the expected “Microsoft Connect Test” in {ms} ms.',
    'net.http.captive':
      'NCSI HTTP probe: {url} returned HTTP {status} instead of the expected answer[[, redirect to {location}]]. This is how a captive portal (a network sign-in page) answers.',
    'net.http.fail': 'NCSI HTTP probe: the request to {url} failed.',
    'net.verdict.online': 'Connected to the internet: the NCSI HTTP probe got the exact expected answer.',
    'net.verdict.captive':
      'The network intercepts HTTP requests: internet access probably needs a captive portal sign-in.',
    'net.verdict.offline': 'No internet connection: no working adapter, or no remote host answers.',
    'net.verdict.dns_broken': 'Remote hosts are reachable by IP address, but DNS does not work.',
    'net.verdict.limited':
      'Limited connection: hosts are reachable by IP address and DNS works, but the NCSI HTTP probe failed (a proxy or HTTP filtering).',

    'inv.av.product':
      'Antivirus “{name}”: {state}, {signature} (source: {source})[[, path {path}]][[, state as of {timestamp}]].',
    'inv.fw.product': 'Third-party firewall “{name}”: {state} (source: {source}).',
    'inv.wsc.unavailable':
      'The Windows Security Center API (WSC) is unavailable, the data is read through WMI.',
    'inv.wmi.unavailable': 'WMI root\\SecurityCenter2 is unavailable too: no Security Center product list.',
    'inv.winfw.service': 'Windows Firewall service (MpsSvc): {state}.',
    'inv.winfw.unavailable': 'Could not read the Windows Firewall policy (INetFwPolicy2).',
    'inv.winfw.profile':
      'Windows Firewall, {profile} profile ({active}): {enabled}, inbound default: {inbound}, outbound: {outbound}.',
    'inv.defender.absent':
      'Microsoft Defender is absent (no WinDefend service or no Defender WMI namespace).',
    'inv.defender.unavailable':
      'WinDefend service: {service}, but the state of Microsoft Defender could not be read.',
    'inv.defender.status':
      'Microsoft Defender: WinDefend service {service}, real-time protection {realtime}, antivirus {antivirus}[[, {mode}]][[, signatures {signature_version}]][[, signature age {signature_age} days]].',
    'inv.files.unavailable': 'Could not read {evidence} to look for security product traces.',
    'inv.files.found': 'Trace of {vendor} ({kind}): {evidence} {value}.',
    'inv.files.none': 'No traces of known antivirus or firewall products in folders, processes or services.',
    'inv.verdict.ok': 'An antivirus and a firewall are installed and on.',
    'inv.verdict.outdated':
      'An antivirus and a firewall are present, but signatures are out of date or protection is snoozed.',
    'inv.verdict.no_av': 'No antivirus that is on was found.',
    'inv.verdict.no_fw': 'No firewall that is on was found.',
    'inv.verdict.none': 'Neither an antivirus nor a firewall that is on was found.',
    'inv.verdict.unknown':
      'Could not establish whether protection is present: the data sources are unavailable.',

    'fw.state.unavailable': 'Could not read the firewall state.',
    'fw.state.service': 'Windows Firewall service (MpsSvc): {state}.',
    'fw.state.enforcing': 'Windows Firewall is on for the active network profiles: {profiles}.',
    'fw.state.off': 'Windows Firewall is off for the active network profile “{profile}”.',
    'fw.state.third_party': 'Third-party firewall “{name}” according to Security Center: {state}.',
    'fw.policy.none': 'Policy test skipped: no resources the policy must block are set in the settings.',
    'fw.policy.reachable': 'Resource {target}, forbidden by the policy, is reachable: answer in {ms} ms.',
    'fw.policy.blocked': 'Resource {target}, forbidden by the policy, is unreachable: {reason}.',
    'fw.policy.inconclusive':
      'Could not connect to {target}, but the error does not show that the firewall blocked the connection.',
    'fw.rule.third_party':
      'Windows Firewall does not filter traffic, third-party firewall “{name}” is on. The rule test only covers Windows Firewall rules, so the verdict rests on the policy test.',
    'fw.rule.needs_admin':
      'Rule test skipped: it needs administrator rights and the UAC prompt is off in the settings.',
    'fw.rule.uac_declined': 'Rule test skipped: the UAC prompt was declined.',
    'fw.rule.elevate_error': 'Could not start the rule test with administrator rights.',
    'fw.rule.gp_override':
      'Rule test skipped: group policy does not apply local firewall rules, a temporary rule would have no effect.',
    'fw.rule.cleanup': 'Removed mamori-probe-* rules left over from interrupted runs: {count}.',
    'fw.rule.control_failed':
      'Rule test skipped: the control hosts {target} are unreachable before any rule is added.',
    'fw.rule.add_failed': 'Could not add the temporary firewall rule.',
    'fw.rule.added': 'Added the temporary rule {name}: block outbound TCP connections of mamori to {target}.',
    'fw.rule.blocked': 'The connection to {target} is blocked by the rule, refused after {ms} ms.',
    'fw.rule.not_enforced':
      'The connection to {target} succeeded despite the blocking rule: the firewall does not enforce rules.',
    'fw.rule.removed': 'Removed the temporary rule {name}.',
    'fw.rule.remove_failed': 'Could not remove the firewall rule {name}.',
    'fw.rule.restored':
      'With the rule removed the connection to {target} works again, so the block came from the rule.',
    'fw.rule.not_restored':
      'With the rule removed the connection to {target} still fails, so the block is not proven to come from the rule.',
    'fw.verdict.works': 'The firewall works: blocking is confirmed by the rule test or the policy test.',
    'fw.verdict.not_enforced':
      'The firewall does not work: the blocking rule was not enforced or a resource forbidden by the policy is reachable.',
    'fw.verdict.disabled': 'The firewall is off and no test showed any traffic filtering.',
    'fw.verdict.unverified':
      'Firewall operation is not confirmed: the rule test did not run and the policy test showed no blocking.',

    'av.eicar.write_failed': 'Could not write the EICAR test file, the on-access test did not run.',
    'av.eicar.blocked_on_write':
      'The antivirus blocked writing the EICAR test file: Windows refused the write with a threat or access denied error.',
    'av.eicar.removed': 'The antivirus deleted the EICAR test file {seconds} s after it was written.',
    'av.eicar.blocked_on_read':
      'The antivirus blocked opening the EICAR test file {seconds} s after it was written.',
    'av.eicar.altered':
      'The antivirus changed the content of the EICAR test file {seconds} s after it was written.',
    'av.eicar.not_detected':
      'The EICAR test file was still intact {seconds} s after it was written: the antivirus did not react.',
    'av.eicar.cleanup_failed': 'Could not delete the temporary folder {path} with the test file.',
    'av.amsi.detected':
      'AMSI: the AMSI test string was detected as a threat, result {result} (AMSI_RESULT_DETECTED threshold 32768).',
    'av.amsi.not_detected': 'AMSI: the AMSI test string was not detected, result {result} (below 32768).',
    'av.amsi.unavailable': 'AMSI is unavailable, the AMSI test did not run.',
    'av.verdict.works': 'The antivirus works: it detected both the EICAR test file and the AMSI test string.',
    'av.verdict.partial':
      'The antivirus works partly: only one of the two tests, EICAR or AMSI, triggered it.',
    'av.verdict.not_working':
      'The antivirus does not work: it detected neither the EICAR test file nor the AMSI test string.',
    'av.verdict.error': 'The antivirus check could not run: the EICAR test file was not written.',

    'report.protected':
      'Protection works: all checks passed, the antivirus and the firewall proved they work.',
    'report.degraded': 'Protection is only partly confirmed: there are warnings, see the module results.',
    'report.partial': 'Not all checks ran: those that ran passed, but the protection verdict is incomplete.',
    'report.unprotected': 'The computer is not protected: at least one protection check failed.',
    'report.error': 'At least one check could not run because of an error.',
    'report.none': 'No checks have run yet.',

    'app.call_failed': 'The check could not run: the program call returned an error.',
  },

  lines: {
    'common.unsupported_os': 'This check only works on Windows.',
    'common.cancelled':
      'The check was interrupted: it was stopped or 3 minutes ran out. The result is incomplete.',

    'net.adapter.ok': 'I found a connected adapter, “{name}”: IPv4 {ipv4}[[, gateway {gateway}]].',
    'net.adapter.none':
      'I found no connected adapter with a working IPv4 address. I do not count 169.254.x.x.',
    'net.adapter.error': 'I could not list the network adapters (GetAdaptersAddresses).',
    'net.dns.ok': 'I checked DNS: {host} resolves to {addrs} in {ms} ms.',
    'net.dns.fail': 'I checked DNS: {host} does not resolve.',
    'net.ncsi_dns.ok': 'NCSI DNS probe: {host} points to {addr}, as it should.',
    'net.ncsi_dns.mismatch':
      'NCSI DNS probe: {host} points to {addr}, but it should be {want}. Something on the way rewrites DNS answers, captive portals do that.',
    'net.ncsi_dns.fail': 'NCSI DNS probe: {host} does not resolve.',
    'net.icmp.ok': 'I pinged {target}: a reply in {rtt_ms} ms, TTL {ttl}.',
    'net.icmp.local':
      '{target} replied with TTL {ttl} in {rtt_ms} ms. That is how this computer (the TUN adapter of a VPN or proxy) or a device nearby answers, not the remote host, so the reply proves nothing.',
    'net.icmp.fail':
      'Ping to {target} failed. ICMP is often filtered on purpose, so there may still be a connection.',
    'net.tcp.ok': 'I opened a TCP connection to {target} in {ms} ms.',
    'net.tcp.fail': 'I could not open a TCP connection to any of {target}.',
    'net.http.ok': 'I requested {url}: the expected “Microsoft Connect Test” came back in {ms} ms.',
    'net.http.captive':
      'I requested {url} and got HTTP {status} instead of the expected answer[[, with a redirect to {location}]]. That is how a captive portal, a network sign-in page, answers.',
    'net.http.fail': 'The request to {url} failed.',
    'net.verdict.online':
      'Verdict: connected to the internet, the NCSI HTTP probe got the exact expected answer.',
    'net.verdict.captive':
      'Verdict: the network intercepts HTTP requests. It looks like internet access needs a captive portal sign-in.',
    'net.verdict.offline':
      'Verdict: no internet connection. There is no working adapter or no remote host answers.',
    'net.verdict.dns_broken': 'Verdict: remote hosts are reachable by IP address, but DNS does not work.',
    'net.verdict.limited':
      'Verdict: the connection is limited. Hosts are reachable by IP address and DNS works, but the NCSI HTTP probe failed, probably a proxy or HTTP filtering.',

    'inv.av.product':
      'Security Center knows the antivirus “{name}”: {state}, {signature} (source: {source})[[, path {path}]][[, state as of {timestamp}]].',
    'inv.fw.product': 'Security Center knows the third-party firewall “{name}”: {state} (source: {source}).',
    'inv.wsc.unavailable': 'The Security Center API (WSC) did not answer, I read the same data through WMI.',
    'inv.wmi.unavailable':
      'WMI root\\SecurityCenter2 did not answer either, so I have no Security Center product list.',
    'inv.winfw.unavailable': 'I could not read the Windows Firewall policy (INetFwPolicy2).',
    'inv.defender.absent':
      'There is no Microsoft Defender here: no WinDefend service and no Defender WMI namespace.',
    'inv.defender.unavailable':
      'WinDefend service: {service}, but I could not read the state of Microsoft Defender.',
    'inv.files.unavailable': 'I could not read {evidence}, the trace search is incomplete.',
    'inv.files.found': 'I found a trace of {vendor} ({kind}): {evidence} {value}.',
    'inv.files.none':
      'I found no traces of known antivirus or firewall products in folders, processes or services.',
    'inv.verdict.ok': 'Verdict: an antivirus and a firewall are installed and on.',
    'inv.verdict.outdated':
      'Verdict: an antivirus and a firewall are present, but signatures are out of date or protection is snoozed.',
    'inv.verdict.no_av': 'Verdict: I found no antivirus that is on.',
    'inv.verdict.no_fw': 'Verdict: I found no firewall that is on.',
    'inv.verdict.none': 'Verdict: I found neither an antivirus nor a firewall that is on.',
    'inv.verdict.unknown':
      'Verdict: I could not establish whether protection is present, the data sources are unavailable.',

    'fw.state.unavailable': 'I could not read the firewall state.',
    'fw.policy.none': 'I skip the policy test: no resources the policy must block are set in the settings.',
    'fw.policy.reachable': 'I reached {target} although the policy forbids it: an answer in {ms} ms.',
    'fw.policy.blocked': 'I could not reach {target}, which the policy forbids: {reason}.',
    'fw.policy.inconclusive':
      'I could not connect to {target}, but the error does not show that the firewall blocked the connection.',
    'fw.rule.needs_admin':
      'I skip the rule test: it needs administrator rights and the UAC prompt is off in the settings.',
    'fw.rule.uac_declined': 'The UAC prompt was declined, so I skipped the rule test.',
    'fw.rule.elevate_error': 'I could not start the rule test with administrator rights.',
    'fw.rule.gp_override':
      'I skip the rule test: group policy does not apply local firewall rules, my rule would have no effect anyway.',
    'fw.rule.cleanup': 'I removed mamori-probe-* rules left over from interrupted runs: {count}.',
    'fw.rule.control_failed':
      'I skip the rule test: the control hosts {target} are unreachable even before the rule, there is nothing to block.',
    'fw.rule.add_failed': 'I could not add the temporary firewall rule.',
    'fw.rule.added':
      'I added the temporary rule {name}: block outbound TCP connections of mamori to {target}.',
    'fw.rule.not_enforced':
      'The connection to {target} succeeded although the rule forbids it. The firewall does not enforce rules.',
    'fw.rule.removed': 'I removed the temporary rule {name}.',
    'fw.rule.remove_failed': 'I could not remove the firewall rule {name}.',
    'fw.rule.restored':
      'With the rule removed, the connection to {target} works again. So it was the rule that blocked it.',
    'fw.rule.not_restored':
      'With the rule removed, the connection to {target} still fails, so the block is not proven to come from the rule.',
    'fw.verdict.works':
      'Verdict: the firewall works, blocking is confirmed by the rule test or the policy test.',
    'fw.verdict.not_enforced':
      'Verdict: the firewall does not work. The blocking rule was not enforced or a resource forbidden by the policy is reachable.',
    'fw.verdict.disabled': 'Verdict: the firewall is off and no test showed any traffic filtering.',
    'fw.verdict.unverified':
      'Verdict: firewall operation is not confirmed. The rule test did not run and the policy test showed no blocking.',

    'av.eicar.write_failed': 'I could not write the EICAR test file, so the on-access test did not happen.',
    'av.eicar.blocked_on_write':
      'The antivirus did not let me write the EICAR test file: Windows refused the write with a threat or access denied error.',
    'av.eicar.removed': 'I wrote the EICAR test file, and {seconds} s later the antivirus deleted it.',
    'av.eicar.blocked_on_read':
      'I wrote the EICAR test file, and {seconds} s later the antivirus stopped me from opening it.',
    'av.eicar.altered':
      'I wrote the EICAR test file, and {seconds} s later the antivirus changed its content.',
    'av.eicar.not_detected':
      'The EICAR test file sat intact for {seconds} s. The antivirus did not react to it.',
    'av.eicar.cleanup_failed':
      'I could not delete the temporary folder {path}, it is worth deleting by hand.',
    'av.amsi.detected':
      'I passed the official AMSI test string to AMSI: the antivirus detected it as a threat, result {result} (AMSI_RESULT_DETECTED threshold 32768).',
    'av.amsi.not_detected':
      'I passed the official AMSI test string to AMSI, but the antivirus did not detect it: result {result}, below 32768.',
    'av.amsi.unavailable': 'AMSI is unavailable, I did not run the AMSI test.',
    'av.verdict.works':
      'Verdict: the antivirus works, it detected both the EICAR test file and the AMSI test string.',
    'av.verdict.partial': 'Verdict: the antivirus works partly, only one of the two tests triggered it.',
    'av.verdict.not_working':
      'Verdict: the antivirus does not work, it detected neither the EICAR test file nor the AMSI test string.',
    'av.verdict.error': 'Verdict: the antivirus check could not run, the EICAR test file was not written.',
  },
}
