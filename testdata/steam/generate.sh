#!/usr/bin/env bash
# Regenerates the Better RPT Steam fixture tree.
# Run from the repository root: bash testdata/steam/generate.sh
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
OUT="$HERE/fixtures"
P="$OUT/primary"
S="$OUT/secondary"
L="$OUT/legacy"

rm -rf "$OUT"
mkdir -p \
	"$P/appcache" \
	"$P/config" \
	"$P/steamapps/common/Spacewar" \
	"$P/steamapps/common/Native Game/data" \
	"$P/steamapps/common/Proton Game" \
	"$P/steamapps/common/Missing Exe" \
	"$P/steamapps/common/No Launch Config" \
	"$P/steamapps/common/Template Launch" \
	"$P/steamapps/common/Path Escape" \
	"$P/steamapps/common/Proton Test" \
	"$P/steamapps/common/SteamLinuxRuntime_test" \
	"$P/steamapps/common/Orphan Manifest" \
	"$P/compatibilitytools.d/GE-Proton-test" \
	"$P/userdata/11111111/config" \
	"$P/userdata/22222222/config" \
	"$S/steamapps/common/Secondary Native" \
	"$L/steamapps/common/Legacy Game"

write_manifest() {
	local dir="$1" id="$2" name="$3" flags="$4" install_dir="$5"
	cat > "$dir/appmanifest_${id}.acf" <<EOF
"AppState"
{
	"appid"		"${id}"
	"Universe"	"1"
	"name"		"${name}"
	"StateFlags"	"${flags}"
	"installdir"	"${install_dir}"
	"LastUpdated"	"0"
	"SizeOnDisk"	"1024"
	"StagingPath"	""
	"UserConfig"
	{
		"language"	"english"
	}
	"MountedConfig"
	{
		"language"	"english"
	}
}
EOF
}

write_manifest "$P/steamapps" 480 "Spacewar" 4 "Spacewar"
write_manifest "$P/steamapps" 900001 "Native Game" 4 "Native Game"
write_manifest "$P/steamapps" 900002 "Proton Game" 4 "Proton Game"
write_manifest "$P/steamapps" 900003 "Proton Test" 4 "Proton Test"
write_manifest "$P/steamapps" 900004 "SteamLinuxRuntime Test" 4 "SteamLinuxRuntime_test"
write_manifest "$P/steamapps" 900005 "Not Installed Game" 2 "Not Installed Game"
write_manifest "$P/steamapps" 900006 "Missing Exe" 4 "Missing Exe"
write_manifest "$P/steamapps" 900009 "No Launch Config" 4 "No Launch Config"
write_manifest "$P/steamapps" 900010 "Orphan Manifest" 4 "Orphan Manifest"
write_manifest "$P/steamapps" 900011 "Template Launch" 4 "Template Launch"
write_manifest "$P/steamapps" 900012 "Path Escape" 4 "Path Escape"
write_manifest "$S/steamapps" 900007 "Secondary Native" 4 "Secondary Native"
write_manifest "$L/steamapps" 900008 "Legacy Game" 4 "Legacy Game"

cat > "$P/steamapps/libraryfolders.vdf.tmpl" <<'VDF'
"libraryfolders"
{
	"0"
	{
		"path"		"{{PRIMARY}}"
		"label"		""
		"contentid"	"0"
		"totalsize"	"0"
		"update_clean_bytes_tally"	"0"
		"time_last_update_corruption"	"0"
		"apps"
		{
			"480"		"0"
			"900001"	"0"
			"900002"	"0"
			"900003"	"0"
			"900004"	"0"
			"900005"	"0"
			"900006"	"0"
			"900009"	"0"
			"900010"	"0"
			"900011"	"0"
			"900012"	"0"
		}
	}
	"1"
	{
		"path"		"{{SECONDARY}}"
		"label"		""
		"contentid"	"0"
		"totalsize"	"0"
		"update_clean_bytes_tally"	"0"
		"time_last_update_corruption"	"0"
		"apps"
		{
			"900007"	"0"
		}
	}
}
VDF

cat > "$L/steamapps/libraryfolders.vdf.tmpl" <<'VDF'
"libraryfolders"
{
	"TimeNextStatsReport"	"0"
	"ContentStatsID"	"0"
	"1"	"{{LEGACY}}"
}
VDF

cat > "$P/config/config.vdf" <<'VDF'
"InstallConfigStore"
{
	"Software"
	{
		"Valve"
		{
			"Steam"
			{
				"CompatToolMapping"
				{
					"0"
					{
						"name"		"GE-Proton-test"
						"config"	""
						"priority"	"250"
					}
					"900002"
					{
						"name"		"GE-Proton-test"
						"config"	""
						"priority"	"250"
					}
					"900007"
					{
						"name"		"native"
						"config"	""
						"priority"	"250"
					}
				}
			}
		}
	}
}
VDF

cat > "$P/loginusers.vdf" <<'VDF'
"users"
{
	"76561197971376839"
	{
		"AccountName"		"fixture"
		"PersonaName"		"Fixture User"
		"RememberPassword"	"1"
		"MostRecent"		"1"
		"Timestamp"		"1700000000"
		"WantsOfflineMode"	"0"
		"SkipOfflineModeWarning"	"0"
	}
	"76561197982487950"
	{
		"AccountName"		"fixture2"
		"PersonaName"		"Second Fixture User"
		"RememberPassword"	"1"
		"MostRecent"		"0"
		"Timestamp"		"1600000000"
		"WantsOfflineMode"	"0"
		"SkipOfflineModeWarning"	"0"
	}
}
VDF

cat > "$P/registry.vdf" <<'VDF'
"Registry"
{
	"HKCU"
	{
		"Software"
		{
			"Valve"
			{
				"Steam"
				{
					"AutoLoginUser"	"fixture"
					"SteamPID"	"0"
				}
			}
		}
	}
}
VDF

cat > "$P/userdata/11111111/config/localconfig.vdf" <<'VDF'
// Better RPT fixture: never point the app at this tree.
"UserLocalConfigStore"
{
	"Software"
	{
		"Valve"
		{
			"Steam"
			{
				"apps"
				{
					"480"
					{
						"LaunchOptions"		""
					}
					"900001"
					{
						"LaunchOptions"		"cd /opt/other && ./launcher # %command%"
					}
					"900002"
					{
						"LaunchOptions"		""
					}
				}
				"friends"
				{
					"123"	"unrelated"
				}
			}
		}
	}
	"WebStorage"
	{
		"apps"
		{
			"480"
			{
				"LaunchOptions"		"webstorage-decoy"
			}
		}
	}
	"apps"
	{
		"480"
		{
			"LaunchOptions"		"root-decoy"
		}
	}
}
VDF

cat > "$P/userdata/22222222/config/localconfig.vdf" <<'VDF'
"UserLocalConfigStore"
{
	"Software"
	{
		"Valve"
		{
			"Steam"
			{
				"apps"
				{
					"480"
					{
						"LaunchOptions"		"cd /opt/account2 && ./other # %command%"
					}
					"900001"
					{
						"LaunchOptions"		""
					}
				}
			}
		}
	}
}
VDF

cat > "$P/userdata/11111111/config/localconfig_duplicate.vdf" <<'VDF'
"UserLocalConfigStore"
{
	"Software"
	{
		"Valve"
		{
			"Steam"
			{
				"apps"
				{
					"480"
					{
						"LaunchOptions"		"first"
					}
					"480"
					{
						"LaunchOptions"		"second"
					}
				}
			}
		}
	}
}
VDF

cat > "$P/userdata/11111111/config/localconfig_missing.vdf" <<'VDF'
"UserLocalConfigStore"
{
	"Software"
	{
		"Valve"
		{
			"Steam"
			{
				"apps"
				{
					"900001"
					{
						"LaunchOptions"		"unrelated"
					}
				}
			}
		}
	}
}
VDF

cat > "$P/userdata/11111111/config/localconfig_no_launchoptions.vdf" <<'VDF'
"UserLocalConfigStore"
{
	"Software"
	{
		"Valve"
		{
			"Steam"
			{
				"apps"
				{
					"480"
					{
						"cloud"		"enabled"
					}
				}
			}
		}
	}
}
VDF

cat > "$P/userdata/11111111/config/localconfig_malformed.vdf" <<'VDF'
"UserLocalConfigStore"
{
	"Software"
	{
		"Valve"
		{
			"Steam"
			{
				"apps"
				{
					"480"
					{
						"LaunchOptions"		"unterminated
VDF

cat > "$P/compatibilitytools.d/GE-Proton-test/toolmanifest.vdf" <<'VDF'
"manifest"
{
	"version"	"2"
	"commandline"	"/proton %verb%"
	"require_tool_appid"	"900004"
	"use_sessions"	"1"
	"compatmanager_layer_name"	"proton"
}
VDF

cat > "$P/steamapps/common/Proton Test/toolmanifest.vdf" <<'VDF'
"manifest"
{
	"version"	"2"
	"commandline"	"/proton %verb%"
	"require_tool_appid"	"900004"
	"use_sessions"	"1"
	"compatmanager_layer_name"	"proton"
}
VDF

cat > "$P/steamapps/common/SteamLinuxRuntime_test/toolmanifest.vdf" <<'VDF'
"manifest"
{
	"commandline"	"/_v2-entry-point --verb=%verb% --"
	"version"	"2"
	"use_tool_subprocess_reaper"	"1"
	"compatmanager_layer_name"	"container-runtime"
}
VDF

touch \
	"$P/steamapps/common/Spacewar/SteamworksExample.exe" \
	"$P/steamapps/common/Native Game/game.x86_64" \
	"$P/steamapps/common/Proton Game/Game.exe" \
	"$P/steamapps/common/No Launch Config/game.x86_64" \
	"$P/steamapps/common/Template Launch/game.x86_64" \
	"$P/steamapps/common/Path Escape/game.x86_64" \
	"$P/steamapps/common/Orphan Manifest/game.x86_64" \
	"$P/steamapps/common/Proton Test/proton" \
	"$P/steamapps/common/SteamLinuxRuntime_test/_v2-entry-point" \
	"$P/compatibilitytools.d/GE-Proton-test/proton" \
	"$S/steamapps/common/Secondary Native/game.x86_64" \
	"$L/steamapps/common/Legacy Game/game.x86_64"

chmod 0755 \
	"$P/steamapps/common/Spacewar/SteamworksExample.exe" \
	"$P/steamapps/common/Native Game/game.x86_64" \
	"$P/steamapps/common/Proton Game/Game.exe" \
	"$P/steamapps/common/No Launch Config/game.x86_64" \
	"$P/steamapps/common/Template Launch/game.x86_64" \
	"$P/steamapps/common/Path Escape/game.x86_64" \
	"$P/steamapps/common/Orphan Manifest/game.x86_64" \
	"$P/steamapps/common/Proton Test/proton" \
	"$P/steamapps/common/SteamLinuxRuntime_test/_v2-entry-point" \
	"$P/compatibilitytools.d/GE-Proton-test/proton" \
	"$S/steamapps/common/Secondary Native/game.x86_64" \
	"$L/steamapps/common/Legacy Game/game.x86_64"

printf 'fixture data\n' > "$P/steamapps/common/Native Game/data/level1.txt"
printf 'This directory intentionally has no executable.\n' > "$P/steamapps/common/Missing Exe/README.txt"

(
	cd "$ROOT"
	go run ./testdata/steam/gen_appinfo.go -out "$OUT/primary/appcache"
)

echo "Steam fixtures generated in $OUT"
