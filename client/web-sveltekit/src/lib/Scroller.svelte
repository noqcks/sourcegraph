<script lang="ts" context="module">
    export interface Capture {
        scroll: number
    }
</script>

<script lang="ts">
    import { createEventDispatcher } from 'svelte'

    export let margin: number

    export function capture(): Capture {
        return { scroll: scroller.scrollTop }
    }

    export function restore(data?: Capture) {
        if (!data) return
        // The actual content of the scroller might not be available yet when `restore` is called,
        // e.g. when the data is fetched asynchronously. In that case, we retry a few times.
        let maxTries = 10
        window.requestAnimationFrame(function syncScroll() {
            if (scroller && scroller.scrollTop !== data.scroll) {
                scroller.scrollTop = data.scroll
                if (maxTries > 0) {
                    maxTries -= 1
                    window.requestAnimationFrame(syncScroll)
                }
            }
        })
    }

    const dispatch = createEventDispatcher<{ more: void }>()

    let viewport: HTMLElement
    let scroller: HTMLElement

    function handleScroll() {
        const remaining = scroller.scrollHeight - (scroller.scrollTop + viewport.clientHeight)

        if (remaining < margin) {
            dispatch('more')
        }
    }
</script>

<div class="viewport" bind:this={viewport}>
    <div class="scroller" bind:this={scroller} on:scroll={handleScroll}>
        <slot />
    </div>
</div>

<style lang="scss">
    .viewport {
        width: 100%;
        height: 100%;
        overflow: hidden;
    }

    .scroller {
        width: 100%;
        height: 100%;
        overflow-y: auto;
        overscroll-behavior-y: contain;
    }
</style>
