
$rules = @(
  @{ id="r1"; name="Entrance Count"; enabled=$true; source="event"; scope=@{cameraId="cam-test"}; condition=@{expression="window.entranceCount >= 3"}; window=@{durationSeconds=120}; trigger=@{mode="rising"; cooldownSeconds=10}; action=@{type="r1_trigger"} },
  @{ id="r2"; name="Occupancy"; enabled=$true; source="state"; scope=@{cameraId="cam-test"; roiId="roi-occ"}; condition=@{expression="occupancy.person > 7"}; trigger=@{mode="rising"; cooldownSeconds=10}; action=@{type="r2_trigger"} },
  @{ id="r3"; name="Dwell"; enabled=$true; source="state"; scope=@{cameraId="cam-test"; roiId="roi-dwell"}; condition=@{expression="dwell.person.max > 120"}; trigger=@{mode="rising"; cooldownSeconds=10}; action=@{type="r3_trigger"} },
  @{ id="r4"; name="Sustained Queue"; enabled=$true; source="state"; scope=@{cameraId="cam-test"; roiId="roi-sustained"}; condition=@{expression="occupancy.person > 4"}; sustain=@{durationSeconds=180}; trigger=@{mode="rising"; cooldownSeconds=10}; action=@{type="r4_trigger"} },
  @{ id="r6"; name="No Staff"; enabled=$true; source="state"; scope=@{cameraId="cam-test"; roiId="roi-staff"}; condition=@{expression="occupancy.person > 5 && staffCount == 0"}; trigger=@{mode="rising"; cooldownSeconds=10}; action=@{type="r6_trigger"} }
)

foreach ($r in $rules) {
  $json = $r | ConvertTo-Json -Depth 5
  $json | Set-Content "temp_rule.json"
  Invoke-RestMethod -Uri http://localhost:8080/api/v1/rules -Method POST -ContentType "application/json" -InFile "temp_rule.json" | Out-Null
}
Write-Host "Rules created successfully"

