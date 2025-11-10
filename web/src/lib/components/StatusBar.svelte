<script lang="ts">
  import { cn } from "$lib/helpers";
  import { STATUS_OK } from "$lib/types/Status";
  
  let { status = STATUS_OK } = $props();
</script>

<div class={cn("w-full p-2 flex justify-center items-center", {
  "bg-green-500 text-white": status.level === 0,
  "bg-yellow-500 text-black": status.level === 1,
  "bg-red-500 text-white": status.level === 2,
})}>
  {status.message}
</div>