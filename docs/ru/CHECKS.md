Русский | [English](../en/CHECKS.md)

# Проверки

Четыре проверки и сводка. Устройство приложения: [ARCHITECTURE.md](ARCHITECTURE.md), пределы
проверок: [LIMITATIONS.md](LIMITATIONS.md).

| Проверка | Пакет | Что выясняет | Права администратора | Что меняет в системе |
|---|---|---|---|---|
| `internet` | `internal/netcheck` | есть ли рабочее подключение к интернету, а если нет, то что мешает | не нужны | ничего |
| `inventory` | `internal/secprod` | какие антивирусы (АВ) и межсетевые экраны (МЭ) есть и в каком они состоянии | не нужны | ничего, только чтение |
| `firewall` | `internal/fwprobe` | фильтрует ли МЭ трафик на самом деле | только для проверки правилом | временное правило брандмауэра Windows |
| `antivirus` | `internal/avprobe` | реагирует ли АВ на тестовые образцы | не нужны | временный файл в `%TEMP%` |

Общие коды:

| Код | Статус | Когда |
|---|---|---|
| `common.cancelled` | `skip` | проверку отменили или вышло её время |
| `common.unsupported_os` | `skip` | программа запущена не на Windows |

Сетевые обращения проверок:

| Адрес | Протокол | Проверка |
|---|---|---|
| `www.msftconnecttest.com` | DNS; HTTP, порт 80, `/connecttest.txt` | `internet` |
| `dns.msftncsi.com` | DNS | `internet` |
| `1.1.1.1`, `8.8.8.8`, `77.88.8.8` (Cloudflare, Google, Яндекс) | ICMP echo; TCP 443 | `internet`; `firewall` (контрольный узел) |
| адреса, введённые пользователем | TCP, HTTP(S) | `firewall`, проверка политики |
| прокси из `HTTP_PROXY`, если `NO_PROXY` не исключает узел | HTTP вместо прямого соединения с `www.msftconnecttest.com` | `internet` |
| DNS-серверы системы | DNS: имена выше и имена узлов из списка политики, через системный резолвер | `internet`, `firewall` |

Других сетевых обращений проверки не делают.

## internet: подключение к интернету

### Что и почему

Адаптер с адресом ещё не значит, что есть интернет. Самое сильное доказательство: точный ответ
сервера проверки Microsoft NCSI (служба, по которой Windows сама определяет подключение) на
обычный HTTP-запрос. Он приходит только при рабочем пути до интернета. Портал авторизации
отвечает на этот запрос сам: перенаправлением, своей страницей или кодом 511. Остальные пробы
различают причины: нет адаптера, сломан DNS, нет маршрута.

### Пробы

Все пробы идут параллельно, у каждой тайм-аут 3 с, вся проверка занимает около одного тайм-аута.

| Проба | Источник | Цель |
|---|---|---|
| адаптеры | `GetAdaptersAddresses` (iphlpapi), только IPv4 | локальные интерфейсы |
| DNS | системный резолвер | `www.msftconnecttest.com` |
| DNS NCSI | системный резолвер, IPv4 | `dns.msftncsi.com`, ожидается `131.107.255.255` |
| ICMP | `IcmpSendEcho2` (iphlpapi.dll), без прав администратора, 32 байта как у `ping.exe` | `1.1.1.1`, `8.8.8.8`, `77.88.8.8` одновременно |
| TCP | TCP-соединение | `1.1.1.1:443`, `8.8.8.8:443`, `77.88.8.8:443` одновременно |
| HTTP | GET без перехода по перенаправлениям, новое соединение, прокси из переменных окружения | `http://www.msftconnecttest.com/connecttest.txt`, тело `Microsoft Connect Test` |

Подробности:

- адаптер подходит, если он включён, не loopback и имеет IPv4-адрес вне `169.254.0.0/16`
  (такой адрес Windows назначает себе, когда DHCP не ответил);
- три узла разных операторов: один из них может фильтроваться или замедляться по пути;
- ответ ICMP засчитывается, только если он пришёл от опрошенного адреса со статусом `IP_SUCCESS`;
- ответ с TTL 255 не прошёл ни одного маршрутизатора: за публичный адрес ответил этот компьютер
  (TUN-адаптер VPN или прокси-клиента) или устройство в том же сегменте. Такой ответ даёт
  `net.icmp.local`, а не `net.icmp.ok`;
- HTTP: код 200 и тело точно `Microsoft Connect Test` дают `net.http.ok`; код 400 и выше,
  кроме 511, даёт `net.http.fail`; всё остальное (перенаправление, другое тело, 204, 511) даёт
  `net.http.captive`. Читаются первые 512 байт тела.

### Находки

| Код | Статус | Параметры | Когда |
|---|---|---|---|
| `net.adapter.ok` | `pass` | `name`, `ipv4`, `gateway` | по одной на каждый подходящий адаптер |
| `net.adapter.none` | `fail` | | подходящих адаптеров нет |
| `net.adapter.error` | `error` | `error` | список адаптеров не прочитан |
| `net.dns.ok` | `pass` | `host`, `addrs`, `ms` | имя разрешилось |
| `net.dns.fail` | `fail` | `host`, `error` | имя не разрешилось |
| `net.ncsi_dns.ok` | `pass` | `host`, `addr` | среди ответов есть `131.107.255.255` |
| `net.ncsi_dns.mismatch` | `warn` | `host`, `addr`, `want` | другой адрес: DNS по пути подменяется, как делают порталы |
| `net.ncsi_dns.fail` | `fail` | `host`, `error` | имя не разрешилось |
| `net.icmp.ok` | `pass` | `target`, `rtt_ms`, `ttl` | первый ответ удалённого узла |
| `net.icmp.local` | `warn` | `target`, `rtt_ms`, `ttl` | были только ответы с TTL 255 |
| `net.icmp.fail` | `warn` | `target`, `error` | ответов нет; ICMP часто фильтруют намеренно, поэтому не `fail` |
| `net.tcp.ok` | `pass` | `target`, `ms` | первое установленное соединение |
| `net.tcp.fail` | `fail` | `target`, `error` | ни одно соединение не установлено |
| `net.http.ok` | `pass` | `url`, `ms` | точный ответ NCSI |
| `net.http.captive` | `warn` | `url`, `status`, `location` | ответ, похожий на портал авторизации |
| `net.http.fail` | `fail` | `url`, `error` | сетевая ошибка или код 400 и выше, кроме 511 |

### Вердикт

Функция `netcheck.verdict`. Узел считается достигнутым, если есть `net.icmp.ok`, или есть
`net.tcp.ok` и нет `net.icmp.local`: TUN, который отвечает на ping за публичные адреса, сам
принимает и TCP-соединения.

| Порядок | Условие | Статус | Код |
|---|---|---|---|
| 1 | есть `net.http.ok` | `pass` | `net.verdict.online` |
| 2 | есть `net.http.captive` | `warn` | `net.verdict.captive` |
| 3 | есть `net.adapter.none` или узел не достигнут | `fail` | `net.verdict.offline` |
| 4 | нет `net.dns.ok` | `warn` | `net.verdict.dns_broken` |
| 5 | иначе | `warn` | `net.verdict.limited` |

Примеры из табличных тестов:

| Находки | Вердикт |
|---|---|
| ICMP fail, TCP ok, HTTP ok | `online` |
| нет адаптера, DNS fail, TCP fail, HTTP ok | `online`: HTTP перевешивает остальное |
| ICMP local, TCP ok, HTTP ok | `online` |
| DNS NCSI mismatch, TCP ok, HTTP captive | `captive` |
| DNS fail, ICMP ok, TCP ok, HTTP fail | `dns_broken` |
| DNS ok, ICMP fail, TCP ok, HTTP fail | `limited` |
| ошибка списка адаптеров, DNS ok, TCP ok, HTTP fail | `limited` |
| нет адаптера, DNS ok, ICMP ok, TCP ok, HTTP fail | `offline` |
| ICMP local, TCP ok, HTTP fail | `offline`: отвечает сам TUN |
| DNS ok, ICMP fail, TCP fail, HTTP fail | `offline`: DNS ответил из локального кэша |

## inventory: наличие АВ и МЭ

### Что и почему

Проверка выясняет, какие АВ и МЭ Windows считает установленными и включёнными, и в каком
состоянии брандмауэр Windows и Microsoft Defender. Это доказательство регистрации и
состояния, а не работы: работу доказывают проверки `firewall` и `antivirus`. Проверка ничего
не меняет и не требует прав администратора: службы открываются с минимальными правами
(`SC_MANAGER_CONNECT` и `SERVICE_QUERY_STATUS`, `SC_MANAGER_ENUMERATE_SERVICE`).

### Источники

| Источник | API | Что читается |
|---|---|---|
| Центр безопасности, основной | COM `IWSCProductList` (wscapi), CLSID `{17072F7B-9ABE-4A74-A261-1EB76B55107A}`, IID `{722A338C-6E8E-4E72-AC27-1417FB0C81C2}`; `Initialize` с `WSC_SECURITY_PROVIDER_ANTIVIRUS` (0x4) и `WSC_SECURITY_PROVIDER_FIREWALL` (0x1), `get_Count`, `get_Item`; у `IWscProduct`: `get_ProductName`, `get_ProductState`, `get_SignatureStatus`, `get_RemediationPath`, `get_ProductStateTimestamp` | АВ и МЭ, зарегистрированные в Windows |
| Центр безопасности, запасной | WMI `root\SecurityCenter2`, классы `AntiVirusProduct`, `FirewallProduct`: `displayName`, `productState`, `pathToSignedProductExe`, `timestamp` | то же; `productState` разбирается эвристикой |
| Брандмауэр Windows | служба `mpssvc` (`QueryServiceStatus`); COM `HNetCfg.FwPolicy2` (`INetFwPolicy2`): `CurrentProfileTypes`, `FirewallEnabled`, `DefaultInboundAction`, `DefaultOutboundAction` для профилей domain (0x1), private (0x2), public (0x4) | действующие настройки профилей |
| Microsoft Defender | служба `WinDefend`; WMI `root\Microsoft\Windows\Defender`, класс `MSFT_MpComputerStatus`: `AMRunningMode`, `AntivirusEnabled`, `RealTimeProtectionEnabled`, `AntivirusSignatureAge`, `AntivirusSignatureVersion` | состояние Defender независимо от Центра безопасности |
| Следы на диске | папки (`registry.ExpandString`, `os.Stat`), процессы (`CreateToolhelp32Snapshot`), службы (`EnumServicesStatusEx`) | таблица известных продуктов |

Правила чтения:

- оба списка (АВ и МЭ) берутся из одного источника: если WSC не ответил хотя бы для одного вида,
  оба берутся из WMI. Отчёт не смешивает источники;
- брандмауэр Windows убирается из списков Центра безопасности: он читается из своей службы и
  политики, а в списке выглядел бы сторонним МЭ. WSC отдаёт его под локализованным именем
  («Брандмауэр Windows» в русской Windows), поэтому он опознаётся по пути исправления
  `%windir%\system32\firewall.cpl`; для WMI, где пути нет, по точным именам `Windows Firewall`,
  `Windows Defender Firewall`, `Microsoft Defender Firewall` и их русским вариантам, без учёта
  регистра;
- профили читаются через `INetFwPolicy2`, а не через WMI `MSFT_NetFirewallProfile`: WMI-класс
  возвращает «не настроено» для значений по умолчанию, `INetFwPolicy2` возвращает действующие;
- если служба `mpssvc` не работает, ошибка чтения профилей не считается ошибкой: без службы
  брандмауэр ничего не фильтрует;
- брандмауэр Windows фильтрует (`Enforcing`), если служба `mpssvc` работает, есть хотя бы один
  активный профиль и все активные профили включены;
- Defender защищает, если служба `WinDefend` работает, включены защита в реальном времени и
  антивирус, и он не в пассивном режиме. Пассивный режим: `AMRunningMode` содержит `passive`
  (`Passive Mode`, `SxS Passive Mode`) или `EDR Block`.

### Состояние продукта

WSC возвращает перечисления из `iwscapi.h`:

| Значение | Состояние (`WSC_SECURITY_PRODUCT_STATE`) | Сигнатуры (`WSC_SECURITY_SIGNATURE_STATUS`) |
|---|---|---|
| 0 | `on` | `out_of_date` |
| 1 | `off` | `up_to_date` |
| 2 | `snoozed` | `unknown` |
| 3 | `expired` | `unknown` |
| другое | `unknown` | `unknown` |

Если WSC не вернул статус сигнатур, он `unknown`. В WMI `productState` разбирается эвристикой:
Microsoft не документирует формат поля.

| Биты | Значение | Результат |
|---|---|---|
| 12-15 (`& 0xF000`) | 0x0, 0x1, 0x2, 0x3, другое | `off`, `on`, `snoozed`, `expired`, `unknown` |
| 4-7 (`& 0x00F0`) | 0x0, 0x1, другое | `up_to_date`, `out_of_date`, `unknown` |

Примеры: `0x061100` (397568) Defender `on`; `0x062100` (401664) `snoozed`, так Defender выглядит
при остановленной службе; `0x061110` `on`, сигнатуры устарели; `0x041000` сторонний продукт `on`.

### Находки

| Код | Статус | Параметры |
|---|---|---|
| `inv.wsc.unavailable` | `warn` | `error`: WSC не ответил, списки из WMI |
| `inv.wmi.unavailable` | `warn` | `error`: WMI тоже не ответил, списков продуктов нет |
| `inv.av.product` | по таблице ниже | `name`, `state`, `signature`, `source` (`wsc` или `wmi`), `path`, `timestamp` |
| `inv.fw.product` | по таблице ниже | `name`, `state`, `source` |
| `inv.winfw.service` | `pass`, если `running`, иначе `fail` | `state`: `running`, `stopped`, `start_pending`, `stop_pending`, `continue_pending`, `pause_pending`, `paused`, `unknown`, `missing` |
| `inv.winfw.unavailable` | `warn` | `error` |
| `inv.winfw.profile` | `pass` включён; `fail` выключен и активен; `warn` выключен и не активен | `profile`, `enabled`, `active`, `inbound`, `outbound` (`block`, `allow`, `unknown`) |
| `inv.defender.status` | по таблице ниже | `realtime`, `antivirus`, `service`, `mode`, `signature_age` (дни), `signature_version` |
| `inv.defender.absent` | `skip` | `error`: нет службы `WinDefend` или пространства имён WMI (`WBEM_E_INVALID_NAMESPACE`, 0x8004100E) |
| `inv.defender.unavailable` | `warn` | `service` (`unknown`, если состояние службы не удалось запросить), `error`: службу `WinDefend` не удалось запросить, или она работает, а `MSFT_MpComputerStatus` не прочитан; о защите ничего не утверждается |
| `inv.files.found` | `pass` | `vendor`, `kind` (`av`, `fw`, `suite`), `evidence` (`path`, `process`, `service`), `value` |
| `inv.files.none` | `warn` | следов известных продуктов нет |
| `inv.files.unavailable` | `warn` | `evidence` (`process` или `service`), `error` |

Статус продукта Центра безопасности:

| Состояние | АВ | МЭ |
|---|---|---|
| `on`, сигнатуры `up_to_date` | `pass` | `pass` |
| `on`, сигнатуры `out_of_date` или `unknown` | `warn` | `pass`: у МЭ нет сигнатур |
| `snoozed`, `unknown` | `warn` | `warn` |
| `off`, `expired` | `fail` | `fail` |

Статус Defender (первое подходящее условие):

| Условие | Статус |
|---|---|
| пассивный режим | `warn` |
| защищает | `pass` |
| не защищает, но включён другой АВ (не Defender) | `warn`: Defender уступает место другому АВ |
| иначе | `fail` |

Если служба `WinDefend` не работает и WMI Defender не отвечает, состояние берётся только по
службе: остановленный Defender ничего не защищает.

### Следы продуктов

Находки `inv.files.found` информационные и на вердикт не влияют. Пути начинаются с переменных
`%ProgramFiles%`, `%ProgramFiles(x86)%`, `%ProgramData%`, имена процессов сравниваются без учёта
регистра, имена служб сопоставляются с шаблонами `path.Match` в нижнем регистре. Вид `suite`
означает, что в некоторых редакциях продукта есть МЭ.

| Продукт | Вид | Папки | Процессы | Службы |
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

### Вердикт

Функция `secprod.verdict` сначала определяет уровень АВ и уровень МЭ: `ok`, `degraded`,
`unknown`, `none`.

Уровень АВ:

| Условие | Уровень |
|---|---|
| Центр безопасности не прочитан, Defender защищает | `ok` |
| Центр безопасности не прочитан, иначе | `unknown` |
| есть АВ `on` с сигнатурами `up_to_date` | `ok` |
| есть АВ `on` с иными сигнатурами или `snoozed` | `degraded` |
| есть АВ `unknown` | `unknown` |
| АВ нет или все `off`, `expired` | `none` |

Записи Defender в списке Центра безопасности пропускаются, если служба `WinDefend` известна и
не работает: Центр безопасности продолжает показывать Defender после остановки службы.
Приоритет: `ok`, затем `degraded`, затем `unknown`.

Уровень МЭ:

| Условие | Уровень |
|---|---|
| брандмауэр Windows фильтрует | `ok` |
| есть сторонний МЭ `on` | `ok` |
| есть сторонний МЭ `snoozed` | `degraded` |
| есть сторонний МЭ `unknown` | `unknown` |
| ничего из этого, но брандмауэр Windows или Центр безопасности не прочитан | `unknown` |
| иначе | `none` |

| Порядок | Условие | Статус | Код |
|---|---|---|---|
| 1 | АВ `none` и МЭ `none` | `fail` | `inv.verdict.none` |
| 2 | АВ `none` | `fail` | `inv.verdict.no_av` |
| 3 | МЭ `none` | `fail` | `inv.verdict.no_fw` |
| 4 | АВ или МЭ `unknown` | `error` | `inv.verdict.unknown` |
| 5 | АВ или МЭ `degraded` | `warn` | `inv.verdict.outdated` |
| 6 | иначе | `pass` | `inv.verdict.ok` |

Примеры из табличных тестов:

| Ситуация | Вердикт |
|---|---|
| АВ `on`, брандмауэр Windows фильтрует | `ok` |
| АВ `on`, сторонний МЭ `on`, служба брандмауэра остановлена | `ok` |
| Defender `off` и другой АВ `on` | `ok` |
| сигнатуры АВ устарели | `outdated` |
| АВ `unknown` рядом с АВ `snoozed` | `outdated` |
| только отключённый Defender | `no_av` |
| Defender `on` в Центре безопасности, служба `WinDefend` остановлена | `no_av` |
| Defender остановлен, другой АВ `on` | `ok` |
| активный профиль брандмауэра выключен | `no_fw` |
| служба брандмауэра остановлена, сторонний МЭ `unknown` | `unknown` |
| Центр безопасности не прочитан, Defender защищает | `ok` |
| Центр безопасности не прочитан, Defender нет или он не прочитан | `unknown` |
| брандмауэр Windows не прочитан, сторонний МЭ `on` | `ok` |

## firewall: работа МЭ

### Что и почему

Включённый МЭ ещё не значит, что он фильтрует трафик. Проверка идёт в три шага:

1. состояние: брандмауэр Windows и сторонние МЭ из Центра безопасности;
2. проверка политики: попытка достучаться до ресурсов, которые политика должна запрещать
   (список задаёт пользователь);
3. проверка правилом: временное правило блокирует соединение самой программы с контрольным
   узлом. Соединение работает до правила, не проходит с правилом и снова работает после его
   удаления. Изменение вызвано правилом, значит МЭ применяет свои правила.

### Шаг 1. Состояние

Источники те же, что у `inventory`: служба `mpssvc` и `INetFwPolicy2`, сторонние МЭ из WSC, при
ошибке из WMI `root\SecurityCenter2`.

| Код | Статус | Параметры | Когда |
|---|---|---|---|
| `fw.state.enforcing` | `pass` | `profiles` | брандмауэр Windows фильтрует, перечислены активные профили |
| `fw.state.off` | `fail` | `profile` | активный профиль выключен, по одной на профиль |
| `fw.state.service` | `fail` | `state` | служба `mpssvc` не работает или отсутствует (`missing`) |
| `fw.state.unavailable` | `warn` | `error` | состояние не прочитано: `windows firewall: ...`, `windows firewall: no active profile` или `security center: ...` |
| `fw.state.third_party` | `pass` для `on`, `warn` для `unknown`, иначе `fail` | `name`, `state` | сторонний МЭ из Центра безопасности |

### Шаг 2. Проверка политики

Адреса задаются полем `policyTargets` интерфейса или флагом `-policy`. Все адреса проверяются
параллельно, тайм-аут 5 с на адрес.

| Запись | TCP-соединение | HTTP |
|---|---|---|
| `host` | `host:443` | нет |
| `host:port` | `host:port` | нет |
| IPv6, в том числе в `[]` | `[адрес]:443` | нет |
| `http://host/путь` | `host:80` или указанный порт | GET по записи |
| `https://host/путь` | `host:443` или указанный порт | GET по записи |

Каждое соединение новое (`net.Dialer`). Для URL после соединения выполняется GET без прокси,
без перехода по перенаправлениям и без повторного использования соединений: ответ должен прийти
от самого адреса.

Классификация (функция `policyFinding`):

| Результат | Находка |
|---|---|
| соединение установлено, для URL получен любой ответ HTTP | `fw.policy.reachable` (`fail`) `{target, ms}` |
| соединение отклонено локально, `WSAEACCES` (10013) | `fw.policy.blocked` (`pass`) `{target, error, reason: wsaeacces}` |
| тайм-аут соединения (не DNS), контрольный узел доступен | `fw.policy.blocked` (`pass`) `{target, error, reason: timeout}` |
| тайм-аут, контрольные узлы недоступны | `fw.policy.inconclusive` (`warn`) `{target, error}` |
| отказ, сброс, недостижимость, ошибка DNS, отмена, неверная запись | `fw.policy.inconclusive` (`warn`) |
| ошибка HTTP после установленного соединения | `fw.policy.inconclusive` (`warn`) |
| список пуст | `fw.policy.none` (`skip`) |

Почему так:

- `WSAEACCES` возвращает локальная платформа фильтрации Windows, это доказательство само по себе;
- отказ или сброс может прийти от удалённой стороны или любого узла по пути;
- после установленного TCP-соединения МЭ уже выпустил поток, поэтому последующая ошибка HTTP
  ничего не доказывает;
- тайм-аут доказывает фильтрацию, только если сеть в этот момент работает. Это проверяется
  одним контрольным раундом за запуск: TCP к `1.1.1.1:443`, `8.8.8.8:443`, `77.88.8.8:443`
  параллельно, 4 с, раунд выполняется только при первом тайм-ауте.

### Шаг 3. Проверка правилом

Где и выполняется ли:

| Условие | Действие |
|---|---|
| брандмауэр Windows не фильтрует или не прочитан, а сторонний МЭ `on` | `fw.rule.third_party` (`warn`) `{name}`, проверка не выполняется: правила стороннего МЭ не являются правилами брандмауэра Windows |
| процесс уже повышен | проверка в этом процессе |
| окно, `allowElevation` | повышенная копия через UAC ([ARCHITECTURE.md](ARCHITECTURE.md#повышение-прав)) |
| иначе | `fw.rule.needs_admin` (`skip`) |

Последовательность (`fwprobe.ruleTest`), объекты COM `HNetCfg.FwPolicy2` и `HNetCfg.FWRule`:

1. Удаляются оставшиеся правила с префиксом `mamori-probe-` от прерванных запусков; если такие
   были, находка `fw.rule.cleanup {count}`.
2. Если `LocalPolicyModifyState` равен `NET_FW_MODIFY_STATE_GP_OVERRIDE` (1), групповая политика
   не применяет локальные правила: `fw.rule.gp_override` (`skip`), проверка останавливается. Если
   значение не прочитано, проверка идёт дальше.
3. Контрольный узел: TCP к `1.1.1.1:443`, `8.8.8.8:443`, `77.88.8.8:443` параллельно, 4 с,
   выбирается первый ответивший. Ни один не ответил: `fw.rule.control_failed` (`skip`).
4. Добавляется правило (`Rules.Add`):

   | Свойство | Значение |
   |---|---|
   | `Name` | `mamori-probe-` и 8 случайных шестнадцатеричных цифр |
   | `Description` | `Temporary rule of the mamori firewall test, removed right after the test` |
   | `Grouping` | `mamori` |
   | `ApplicationName` | путь к выполняемому `mamori.exe` |
   | `Protocol` | 6 (TCP) |
   | `RemoteAddresses` | IP контрольного узла |
   | `RemotePorts` | порт контрольного узла (443) |
   | `Direction` | 2 (исходящее) |
   | `Action` | 0 (блокировать) |
   | `Profiles` | 0x7FFFFFFF (все) |
   | `Enabled` | true |

5. Пауза 300 мс, новое соединение с контрольным узлом за 4 с: ошибка даёт `fw.rule.blocked`,
   успех даёт `fw.rule.not_enforced`.
6. Правило удаляется: `fw.rule.removed` или `fw.rule.remove_failed`.
7. Только если был блок и правило удалено: пауза 300 мс, новое соединение: успех даёт
   `fw.rule.restored`, ошибка даёт `fw.rule.not_restored`.

Каждое соединение после добавления правила новое: открытое ранее соединение обошло бы правило.
Без шага 7 случайный обрыв сети выглядел бы как блок.

| Код | Статус | Параметры |
|---|---|---|
| `fw.rule.cleanup` | `pass` | `count` |
| `fw.rule.gp_override` | `skip` | |
| `fw.rule.control_failed` | `skip` | `target`, `error` |
| `fw.rule.added` | `pass` | `name`, `target` |
| `fw.rule.add_failed` | `error` | `error`; также при ошибке `os.Executable`, инициализации COM, открытия политики |
| `fw.rule.blocked` | `pass` | `target`, `error`, `ms` |
| `fw.rule.not_enforced` | `fail` | `target` |
| `fw.rule.removed` | `pass` | `name` |
| `fw.rule.remove_failed` | `error` | `name` (`mamori-probe-*`, если не прочитан список), `error` |
| `fw.rule.restored` | `pass` | `target` |
| `fw.rule.not_restored` | `warn` | `target`, `error` |
| `fw.rule.third_party` | `warn` | `name` |
| `fw.rule.needs_admin` | `skip` | |
| `fw.rule.uac_declined` | `skip` | |
| `fw.rule.elevate_error` | `error` | `error` |

### Вердикт

Функция `fwprobe.verdict`. Обозначения:

- МЭ включён: есть находка `fw.state.*` со статусом `pass`;
- МЭ выключен: есть находка `fw.state.*` со статусом `fail`;
- правило доказано: есть `fw.rule.blocked` и `fw.rule.restored`;
- политика доказана: есть `fw.policy.blocked` с `reason: wsaeacces`, или с любой причиной
  при включённом МЭ (без включённого МЭ тихий сброс у провайдера или мёртвый узел выглядит так же).

| Порядок | Условие | Статус | Код |
|---|---|---|---|
| 1 | есть `fw.rule.not_enforced` или `fw.policy.reachable` | `fail` | `fw.verdict.not_enforced` |
| 2 | правило доказано или политика доказана | `pass` | `fw.verdict.works` |
| 3 | МЭ выключен и не включён | `fail` | `fw.verdict.disabled` |
| 4 | иначе | `warn` | `fw.verdict.unverified` |

Повышенная копия (`--probe firewall-rule`) решает свой вердикт отдельно (`ruleVerdict`):
`fw.rule.not_enforced` даёт `fail`, доказанное правило `pass`, остальное `skip` с кодом
`fw.verdict.unverified`. В окне учитываются только её находки, вердикт решает родитель.

Примеры из табличных тестов:

| Ситуация | Вердикт |
|---|---|
| фильтрует, правило: блок и восстановление | `works` |
| фильтрует, блок без восстановления | `unverified` |
| фильтрует, правило не применилось | `not_enforced` |
| правило работает, но адрес политики доступен | `not_enforced` |
| фильтрует, адрес политики сброшен по тайм-ауту, UAC отклонён | `works` |
| профиль выключен, нет прав | `disabled` |
| профиль выключен, адрес политики сброшен по тайм-ауту | `disabled` |
| профиль выключен, адрес политики отклонён локально (`WSAEACCES`) | `works` |
| только сторонний МЭ, проверка правилом пропущена | `unverified` |
| сторонний МЭ `on`, адрес политики сброшен по тайм-ауту | `works` |
| состояние не прочитано, адрес политики сброшен по тайм-ауту | `unverified` |
| состояние не прочитано, правило доказано | `works` |

### Меры безопасности

- Правило касается только `mamori.exe` и только исходящего TCP к одному IP и порту: трафик
  других программ не затрагивается.
- Уникальное имя с префиксом `mamori-probe-` и случайной частью, группа `mamori`, описание.
- Правило удаляется сразу после проверки, в том числе при отмене (`defer`). Правила прерванных
  запусков удаляются в начале каждой проверки правилом. `scripts/e2e.ps1` проверяет, что правил
  не осталось.
- Права администратора нужны только этому шагу и запрашиваются через UAC по явному выбору
  (`allowElevation`). Шаги 1 и 2 только читают состояние и открывают соединения.

## antivirus: работа АВ

### Что и почему

- EICAR: стандартный тестовый файл, который антивирусы по договорённости обнаруживают как
  вредоносный, хотя он безвреден. Если АВ удаляет, блокирует или меняет файл после записи и
  открытия, работает защита файлов в реальном времени.
- AMSI (Antimalware Scan Interface): интерфейс, через который Windows и программы передают
  содержимое зарегистрированному антивирусу. Обнаружение тестовой строки AMSI показывает, что
  работает проверка через этот интерфейс. Сканирование идёт в памяти, на диск ничего не пишется.

Это два разных пути, поэтому проверяются оба.

### EICAR

1. Создаётся папка `%TEMP%\mamori-av-<случайно>`, в неё пишется `eicar-test.txt` (права
   `0600`). Строка EICAR (68 байт) хранится закодированной и собирается в памяти перед записью,
   буфер затем обнуляется.
2. Ошибка записи с кодом Win32 5, 225 или 226 даёт `av.eicar.blocked_on_write`. Иная ошибка
   (или ошибка создания папки) даёт `av.eicar.write_failed`.
3. Каждые 250 мс файл открывается и читается заново: открытие запускает проверку при доступе.
   Опрос идёт до `EicarWait` (по умолчанию 15 с, поле `eicarWaitSec`, флаг `-eicar-wait`).

   | Результат чтения | Действие |
   |---|---|
   | файла нет | `av.eicar.removed` |
   | ошибка 5, 225, 226 | `av.eicar.blocked_on_read` |
   | содержимое не совпадает с EICAR (сравнение SHA-256) | `av.eicar.altered` |
   | содержимое прежнее | ждать дальше |
   | иная ошибка, например файл занят сканером | ждать дальше |
   | время вышло | `av.eicar.not_detected` |

4. Папка удаляется, до трёх повторов через 250 мс: сканер может ещё держать файл.

| Код Win32 | Имя |
|---|---|
| 5 | `ERROR_ACCESS_DENIED` |
| 225 | `ERROR_VIRUS_INFECTED` |
| 226 | `ERROR_VIRUS_DELETED` |

### AMSI

1. Из `amsi.dll` берутся экспорты `AmsiInitialize`, `AmsiOpenSession`, `AmsiScanString`,
   `AmsiCloseSession`, `AmsiUninitialize`; если какого-то нет, AMSI недоступен.
2. `AmsiInitialize("mamori")`, `AmsiOpenSession`, `AmsiScanString` с тестовой строкой AMSI из
   документации Microsoft (54 байта, хранится закодированной), затем закрытие сессии.
3. Результат `32768` (`AMSI_RESULT_DETECTED`) и выше: обнаружено. Меньше: не обнаружено.
   Ошибочный HRESULT: AMSI недоступен.

AMSI выполняется после EICAR.

### Находки

| Код | Статус | Параметры |
|---|---|---|
| `av.eicar.blocked_on_write` | `pass` | `error` |
| `av.eicar.removed` | `pass` | `seconds` |
| `av.eicar.blocked_on_read` | `pass` | `error`, `seconds` |
| `av.eicar.altered` | `pass` | `seconds` |
| `av.eicar.not_detected` | `fail` | `seconds`, `error` (если последнее чтение завершилось ошибкой) |
| `av.eicar.write_failed` | `error` | `error` |
| `av.eicar.cleanup_failed` | `warn` | `path`, `error` |
| `av.amsi.detected` | `pass` | `result` |
| `av.amsi.not_detected` | `fail` | `result` |
| `av.amsi.unavailable` | `warn` | `error` |

`seconds` с одним знаком после точки.

### Вердикт

Функция `avprobe.verdict`, все девять сочетаний покрыты табличным тестом.

| EICAR | AMSI | Статус | Код |
|---|---|---|---|
| не выполнен (не удалось создать папку или записать файл) | любой | `error` | `av.verdict.error` |
| обнаружен | обнаружен | `pass` | `av.verdict.works` |
| обнаружен | не обнаружен или недоступен | `warn` | `av.verdict.partial` |
| не обнаружен | обнаружен | `warn` | `av.verdict.partial` |
| не обнаружен | не обнаружен или недоступен | `fail` | `av.verdict.not_working` |

### Меры безопасности

- Строки EICAR и AMSI в исходниках и в `.exe` хранятся только закодированными (XOR с ключом,
  затем base64) и собираются в памяти перед использованием. Тест сравнивает только хеши SHA-256.
- Файл EICAR пишется только в свою временную папку и удаляется вместе с ней. Он текстовый
  (`.txt`) и не запускается.
- Тестовая строка AMSI на диск не пишется.

## Сводка (`internal/report`)

`report.Summarize` строит `Summary`: `version`, `commit`, `system`, `generated`, `status`,
`code`, `results`.

Правила `report.verdict`:

- результаты со статусом `skip` не учитываются;
- `fail` проверки `internet` считается `warn`: отсутствие подключения само по себе не делает
  компьютер незащищённым, а рабочее подключение не искупает сломанный МЭ;
- итог: самый тяжёлый статус остальных (`pass` < `skip` < `warn` < `fail` < `error`).

| Условие | Статус | Код |
|---|---|---|
| не выполнилась ни одна проверка | `skip` | `report.none` |
| все `pass`, но часть пропущена | `warn` | `report.partial` |
| все `pass` | `pass` | `report.protected` |
| худший `warn` | `warn` | `report.degraded` |
| худший `error` | `error` | `report.error` |
| худший `fail` | `fail` | `report.unprotected` |

`error` тяжелее `fail`: если одна проверка не прошла, а другая завершилась ошибкой, сводка
будет `report.error`.

Примеры из табличных тестов:

| Результаты | Сводка |
|---|---|
| все четыре `pass` | `protected` |
| `internet` `pass`, `firewall` `skip` | `partial` |
| `internet` `fail`, `firewall` и `antivirus` `pass` | `degraded` |
| `firewall` `fail`, остальные `pass` | `unprotected` |
| `antivirus` `error`, `firewall` `fail` | `error` |
| только `skip` | `none` |
