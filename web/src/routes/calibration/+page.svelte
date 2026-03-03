<script lang="ts">
  import { goto, beforeNavigate } from "$app/navigation";
  import StatusBar from "$lib/components/StatusBar.svelte";
  import {
    cn,
    connectSSE,
    requestData,
    requestSnapshot,
    sendCommand,
  } from "$lib/helpers";
  import { STATUS_OK } from "$lib/types/Status";
  import { onMount, onDestroy } from "svelte";
  import { parse } from "svelte/compiler";

  let status = $state(STATUS_OK);

  let calPoint = $state("");
  let calTable = $state<Record<string, string>>({});
  let load = $state(0.0);
  let sseCleanup: (() => void) | null = null;

  onMount(() => {
    requestData().then((data) => {
      if (data.CalTable) {
        calTable = data.CalTable.CalTable;
        order();
      }
    });

    sseCleanup = connectSSE((m) => {
      load = m["Load"];
      let alarm_err = m["AlarmError"];
      let alarm_warn = m["AlarmWarn"];
      let max_tension = m["MaxTension"];
      let e_stop = m["EStop"];
      let usb_connected = m["UsbConnected"];
      let usb_error = m["UsbError"];
      let log_enabled = m["LogEnabled"];
      let device_log_error = m["DeviceLogError"];
      let control_loop_error = m["ControlLoopError"];

      if (e_stop) {
        status = {
          level: 2,
          message: "ERROR: STOP ENGAGED",
        };
      } else if (alarm_err) {
        if (max_tension === true) {
          status = {
            level: 2,
            message: `ERROR: Load Exceeds Maximum Tension`,
          };
        } else {
          status = {
            level: 2,
            message: `ERROR: Overload Exceeded`,
          };
        }
      } else if (alarm_warn) {
        status = {
          level: 1,
          message: `WARNING: High Load Approaching Maximum Tension`,
        };
      } else if (control_loop_error) {
        status = {
          level: 2,
          message: `ERROR: ${control_loop_error}`,
        };
      } else if (log_enabled && device_log_error) {
        status = {
          level: 2,
          message: `ERROR: ${device_log_error}`,
        };
      } else if (log_enabled && (usb_error || !usb_connected)) {
        status = {
          level: 2,
          message: usb_error ? `ERROR: ${usb_error}` : "ERROR: USB not connected",
        };
      } else {
        status = {
          level: 0,
          message: "System OK",
        };
      }
    });
  });

  onDestroy(() => {
    if (sseCleanup) {
      sseCleanup();
      sseCleanup = null;
    }
  });

  beforeNavigate(() => {
    if (sseCleanup) {
      sseCleanup();
      sseCleanup = null;
    }
  });

  const updateField = (value: number | ".") => {
    calPoint = `${calPoint}${value}`;
  };

  const clearField = () => {
    calPoint = "";
  };

  const setZero = () => {
    calTable["0"] = load.toFixed(3);
    order();
  };

  const setCal = () => {
    if (calPoint) {
      const r = parseFloat(calPoint);
      calTable[r.toString()] = load.toFixed(3); // Replace 0 with actual raw reading
    }
    order();
  };

  const order = () => {
    calTable = Object.fromEntries(
      Object.entries(calTable).sort(
        (a, b) => parseFloat(a[0]) - parseFloat(b[0])
      )
    );
  };

  const removeCalPoint = (knownLoad: string) => {
    delete calTable[knownLoad];
    order();
  };

  const setCalTable = async () => {
    // Save calTable to backend or local storage
    await sendCommand("SetCalibrationTable", { CalTable: calTable });
    window.location.href = "/"
  };
</script>

<div class="w-full h-screen flex flex-col">
  <StatusBar {status} />

  <main
    class={cn("w-full flex-1 flex justify-center items-center p-2 gap-2", {
      "bg-gray-800": status.level === 0,
      "bg-yellow-500": status.level === 1,
      "bg-red-500": status.level === 2,
    })}
  >
    <div class="flex-1 h-full relative flex flex-col">
      <h1>Calibration Settings</h1>
      <div class="flex w-full mt-1 gap-4">
        <div class="flex-1">
          <label for="tension" class="block text-gray-400 text-lg"
            >Live Load (kg):</label
          >
          <div
            class={cn(
              "flex items-center bg-gray-900 p-2 rounded-xl text-4xl justify-end text-green-500 border-2",
              "border-gray-700"
            )}
          >
            {load.toFixed(0)}
          </div>
          <button class="btn w-full mt-2" onclick={setZero}
            >Set Zero Point</button
          >
        </div>
        <div class="flex-1">
          <label for="tension" class="block text-gray-400 text-lg"
            >Known Load (kg):</label
          >
          <div
            class={cn(
              "flex items-center bg-gray-900 p-2 rounded-xl text-4xl justify-end text-blue-500 border-2",
              "border-blue-500"
            )}
          >
            {calPoint || "0"}
          </div>
          <button class="btn w-full mt-2" onclick={setCal}>Set Cal Point</button
          >
        </div>
      </div>

      <div class="h-1 w-full max-w-7xl bg-gray-700 mt-4"></div>

      <h3 class="mt-2 text-sm">Calibration Points</h3>

      <div class="w-full h-34 overflow-y-auto bg-gray-900 mt-2 rounded">
        <table class="table-auto text-white w-full text-sm">
          <thead>
            <tr class="bg-gray-700">
              <td class="p-1">Known Load (kg)</td>
              <td class="p-1">Raw Reading</td>
              <td class="p-1 text-right"></td>
            </tr>
          </thead>

          <tbody>
            {#each Object.entries(calTable) as [knownLoad, rawReading]}
              <tr>
                <td class="p-1">{parseFloat(knownLoad).toFixed(0)}</td>
                <td class="p-1">{parseFloat(rawReading).toFixed(0)}</td>
                <td class="p-1 text-right">
                  <button
                    class="btn danger text-xs p-1 font-normal"
                    onclick={() => removeCalPoint(knownLoad)}>Remove</button
                  >
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
    <div class="flex-1 h-full grid grid-cols-3 grid-rows-4 gap-2">
      <button class="btn text-xl" onclick={() => updateField(1)}>1</button>
      <button class="btn text-xl" onclick={() => updateField(2)}>2</button>
      <button class="btn text-xl" onclick={() => updateField(3)}>3</button>
      <button class="btn text-xl" onclick={() => updateField(4)}>4</button>
      <button class="btn text-xl" onclick={() => updateField(5)}>5</button>
      <button class="btn text-xl" onclick={() => updateField(6)}>6</button>
      <button class="btn text-xl" onclick={() => updateField(7)}>7</button>
      <button class="btn text-xl" onclick={() => updateField(8)}>8</button>
      <button class="btn text-xl" onclick={() => updateField(9)}>9</button>
      <button class="btn text-xl" onclick={() => clearField()}>C</button>
      <button class="btn text-xl" onclick={() => updateField(0)}>0</button>
      <button class="btn text-xl" onclick={() => updateField(".")}>.</button>
    </div>
  </main>

  <footer
    class={cn("w-full p-2 flex gap-2", {
      "bg-gray-800": status.level === 0,
      "bg-yellow-500": status.level === 1,
      "bg-red-500": status.level === 2,
    })}
  >
    <button class="btn gprimary" onclick={setCalTable}> Set & Save </button>
    <button class="btn" onclick={() => window.location.href = "/"}> Back </button>
  </footer>
</div>
