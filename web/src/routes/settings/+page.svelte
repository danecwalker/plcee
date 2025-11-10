<script lang="ts">
  import { beforeNavigate, goto } from "$app/navigation";
  import StatusBar from "$lib/components/StatusBar.svelte";
  import { cn, connectSSE, requestData, sendCommand } from "$lib/helpers";
  import { STATUS_OK } from "$lib/types/Status";
  import { error } from "@sveltejs/kit";
  import { onDestroy, onMount } from "svelte";

  let status = $state(STATUS_OK);

  let logDelayMs = $state("");
  let logIntervalMs = $state("");
  let logEnabled = $state(false);
  let sseCleanup: (() => void) | null = null;

  let selectedField: "logDelayMs" | "logIntervalMs" | "logEnabled" =
    $state("logDelayMs");

  const updateField = (value: number | ".") => {
    if (selectedField === "logDelayMs") {
      logDelayMs = `${logDelayMs}${value}`;
    } else if (selectedField === "logIntervalMs") {
      logIntervalMs = `${logIntervalMs}${value}`;
    } 
  };

  const clearField = () => {
    if (selectedField === "logDelayMs") {
      logDelayMs = "";
    } else if (selectedField === "logIntervalMs") {
      logIntervalMs = "";
    } 
  };

  const save = async () => {
    // parse to floats
    const LogDelayMs = parseInt(logDelayMs);
    const IntervalMs = parseFloat(logIntervalMs);
    const Enabled = logEnabled;
    // Save the tension settings to backend or local storage
    await sendCommand("SetLogSettings", {
      LogDelayMs,
      IntervalMs,
      Enabled,
    });
    goto("/");
  };

  onMount(() => {
    requestData().then((d) => {
      logDelayMs = `${d["LogSettings"]["LogDelayMs"]}`;
      logIntervalMs = `${d["LogSettings"]["IntervalMs"]}`;
      logEnabled = d["LogSettings"]["Enabled"];
    });

    sseCleanup = connectSSE((m) => {
      let alarm_err = m["AlarmError"];
      let alarm_warn = m["AlarmWarn"];
      let max_tension = m["logDelayMs"];
      let e_stop = m["EStop"];

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
    <div class="flex-1 h-full">
      <h1>Log Settings</h1>
      <div>
        <label for="tension" class="block text-gray-400 text-lg"
          >Log Delay (ms):</label
        >
        <div
          onclick={() => {
            selectedField = "logDelayMs";
          }}
          class={cn(
            "flex items-center bg-gray-900 p-2 rounded-xl text-6xl justify-end text-green-500 border-2",
            {
              "border-green-500": selectedField === "logDelayMs",
              "border-gray-700": selectedField !== "logDelayMs",
            }
          )}
        >
          {logDelayMs || "0"}
        </div>
      </div>

      <div>
        <label for="tension" class="block text-gray-400 text-lg"
          >Log Interval (ms):</label
        >
        <div
          onclick={() => {
            selectedField = "logIntervalMs";
          }}
          class={cn(
            "flex items-center bg-gray-900 p-2 rounded-xl text-6xl justify-end text-blue-500 border-2",
            {
              "border-blue-500": selectedField === "logIntervalMs",
              "border-gray-700": selectedField !== "logIntervalMs",
            }
          )}
        >
          {logIntervalMs || "0"}
        </div>
      </div>

      <div>
        <label for="tension" class="block text-gray-400 text-lg"
          >Log Enabled:</label
        >
        <div
          onclick={() => {
            logEnabled = !logEnabled;
          }}
          class={cn(
            "flex items-center bg-gray-900 p-2 rounded-xl text-6xl justify-end text-blue-500 border-2", "border-gray-700"
          )}
        >
          {logEnabled ? "ON" : "OFF"}
        </div>
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
    <button class="btn gprimary" onclick={save}> Set & Save </button>
    <button class="btn" onclick={() => goto("/")}> Back </button>
  </footer>
</div>
