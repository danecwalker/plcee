<script lang="ts">
  import { beforeNavigate, goto } from "$app/navigation";
  import StatusBar from "$lib/components/StatusBar.svelte";
  import { cn, connectSSE, logout, requestData, sendCommand } from "$lib/helpers";
  import { STATUS_OK } from "$lib/types/Status";
  import { error } from "@sveltejs/kit";
  import { onDestroy, onMount } from "svelte";

  let status = $state(STATUS_OK);

  let maxTension = $state("");
  let warningPercent = $state("");
  let errorPercent = $state("");
  let sseCleanup: (() => void) | null = null;

  let selectedField: "maxTension" | "warningPercent" | "errorPercent" =
    $state("maxTension");

  const updateField = (value: number | ".") => {
    if (selectedField === "maxTension") {
      maxTension = `${maxTension}${value}`;
    } else if (selectedField === "warningPercent") {
      warningPercent = `${warningPercent}${value}`;
    } else if (selectedField === "errorPercent") {
      errorPercent = `${errorPercent}${value}`;
    }
  };

  const clearField = () => {
    if (selectedField === "maxTension") {
      maxTension = "";
    } else if (selectedField === "warningPercent") {
      warningPercent = "";
    } else if (selectedField === "errorPercent") {
      errorPercent = "";
    }
  };

  const save = async () => {
    // parse to floats
    const MaxTensionValue = parseFloat(maxTension);
    const WarnTensionPercent = parseFloat(warningPercent);
    const ErrorTensionPercent = parseFloat(errorPercent);
    // Save the tension settings to backend or local storage
    await sendCommand("SetTensionSettings", {
      MaxTensionValue,
      WarnTensionPercent,
      ErrorTensionPercent,
    });
    await logout();
    goto("/");
  };

  onMount(() => {
    requestData().then((d) => {
      maxTension = `${d["TensionSettings"]["MaxTensionValue"]}`;
      warningPercent = `${d["TensionSettings"]["WarnTensionPercent"]}`;
      errorPercent = `${d["TensionSettings"]["ErrorTensionPercent"]}`;
    });

    sseCleanup = connectSSE((m) => {
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
      <h1>Tension Settings</h1>
      <div>
        <label for="tension" class="block text-gray-400 text-lg"
          >Max Tension (kg):</label
        >
        <div
          onclick={() => {
            selectedField = "maxTension";
          }}
          class={cn(
            "flex items-center bg-gray-900 p-2 rounded-xl text-6xl justify-end text-green-500 border-2",
            {
              "border-green-500": selectedField === "maxTension",
              "border-gray-700": selectedField !== "maxTension",
            }
          )}
        >
          {maxTension || "0"}
        </div>
      </div>

      <div class="flex w-full mt-1 gap-4">
        <div class="flex-1">
          <label for="tension" class="block text-gray-400 text-lg"
            >Warning (%):</label
          >
          <div
            onclick={() => {
              selectedField = "warningPercent";
            }}
            class={cn(
              "flex items-center bg-gray-900 p-2 rounded-xl text-4xl justify-end text-blue-500 border-2",
              {
                "border-blue-500": selectedField === "warningPercent",
                "border-gray-700": selectedField !== "warningPercent",
              }
            )}
          >
            {warningPercent || "0"}
          </div>
        </div>
        <div class="flex-1">
          <label for="tension" class="block text-gray-400 text-lg"
            >Error (%):</label
          >
          <div
            onclick={() => {
              selectedField = "errorPercent";
            }}
            class={cn(
              "flex items-center bg-gray-900 p-2 rounded-xl text-4xl justify-end text-blue-500 border-2",
              {
                "border-blue-500": selectedField === "errorPercent",
                "border-gray-700": selectedField !== "errorPercent",
              }
            )}
          >
            {errorPercent || "0"}
          </div>
        </div>
      </div>

      <div class="h-1 w-full max-w-7xl bg-gray-700 mt-4"></div>

      <h2 class="mt-4">Calculated Results</h2>

      <div class="flex flex-col text-gray-400">
        <div class="flex justify-between items-center mt-2">
          <span>Warning At:</span>
          <span
            >= {(parseFloat(maxTension) * parseFloat(warningPercent)) / 100 ||
              0} kg</span
          >
        </div>
        <div class="flex justify-between items-center mt-2">
          <span>Error At:</span>
          <span
            >= {(parseFloat(maxTension) * parseFloat(errorPercent)) / 100 || 0} kg</span
          >
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
    <button class="btn" onclick={() => 
      logout().then(() => goto("/"))
    }> Back </button>
  </footer>
</div>
