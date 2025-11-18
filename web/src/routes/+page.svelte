<script lang="ts">
  import { beforeNavigate, goto } from "$app/navigation";
  import StatusBar from "$lib/components/StatusBar.svelte";
  import { cn, connectSSE, requestSnapshot, sendCommand } from "$lib/helpers";
  import { STATUS_OK } from "$lib/types/Status";
  import { Settings } from "@lucide/svelte";
  import { onDestroy, onMount } from "svelte";

  let status = $state(STATUS_OK);
  let load = $state(0.0);
  let proxValue = $state(0.0);
  let speed = $state(false);
  let sseCleanup: (() => void) | null = null;

  onMount(() => {
    sseCleanup = connectSSE((m) => {
      load = m["Load"];
      proxValue = m["ProxValue"];
      speed = m["Speed"];

      let alarm_err = m["AlarmError"];
      let alarm_warn = m["AlarmWarn"];
      let max_tension = m["MaxTension"];
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

  const setSpeed = (hi: boolean) => {
    sendCommand("SetSpeed", hi);
  };
</script>

<div class="w-full h-screen flex flex-col">
  <StatusBar {status} />

  <main
    class={cn(
      "w-full flex-1 flex flex-col justify-center items-center p-2 font-mono relative",
      {
        "bg-gray-800": status.level === 0,
        "bg-yellow-500": status.level === 1,
        "bg-red-500": status.level === 2,
      }
    )}
  >
    <button
      class="absolute top-4 right-4 text-gray-500 hover:text-white cursor-pointer text-2xl"
      onclick={() => window.location.href = "/settings"}
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="1.5em"
        height="1.5em"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        class="lucide lucide-settings-icon lucide-settings"
        ><path
          d="M9.671 4.136a2.34 2.34 0 0 1 4.659 0 2.34 2.34 0 0 0 3.319 1.915 2.34 2.34 0 0 1 2.33 4.033 2.34 2.34 0 0 0 0 3.831 2.34 2.34 0 0 1-2.33 4.033 2.34 2.34 0 0 0-3.319 1.915 2.34 2.34 0 0 1-4.659 0 2.34 2.34 0 0 0-3.32-1.915 2.34 2.34 0 0 1-2.33-4.033 2.34 2.34 0 0 0 0-3.831A2.34 2.34 0 0 1 6.35 6.051a2.34 2.34 0 0 0 3.319-1.915"
        /><circle cx="12" cy="12" r="3" /></svg
      >
    </button>

    <div class="w-full flex-1 flex justify-center items-center">
      <div class="text-green-500">
        <span class="text-9xl">{load.toFixed(0)}</span>
        <span class="text-4xl font-medium text-gray-500">kg</span>
      </div>
    </div>
    <div class="h-1 w-full max-w-xl bg-gray-700"></div>
    <div class="w-full flex-1 flex justify-center items-center">
      <div class="text-blue-500">
        <span class="text-9xl">{proxValue.toFixed(1)}</span>
        <span class="text-4xl font-medium text-gray-500">m</span>
      </div>
    </div>
  </main>

  <footer
    class={cn("w-full p-2 flex gap-2", {
      "bg-gray-800": status.level === 0,
      "bg-yellow-500": status.level === 1,
      "bg-red-500": status.level === 2,
    })}
  >
    <button class="btn" onclick={() => sendCommand("ZeroDistance")}>
      Zero Distance
    </button>
    <button class="btn" onclick={() => { window.location.href = "/set-tension" }}>
      Set Tension
    </button>
    <button class="btn" onclick={() => window.location.href = "/calibration"}>
      Calibrate
    </button>
    <button
      class={cn("btn", {
        "hold gprimary": !speed,
      })}
      onclick={() => setSpeed(false)}
    >
      Lo Speed
    </button>
    <button
      class={cn("btn", {
        "hold gprimary": speed,
      })}
      onclick={() => setSpeed(true)}
    >
      Hi Speed
    </button>
  </footer>
</div>
