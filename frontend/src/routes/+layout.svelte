<script lang="ts">
	import '../app.css';
	import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query';
	import { onMount } from 'svelte';
	import { connectEventSource } from '$lib/stores/events';
	import { page } from '$app/state';
	import {
		LayoutDashboard,
		ListTodo,
		Code2,
		CalendarClock,
		Webhook,
		Settings,
		Zap
	} from '@lucide/svelte';

	let { children } = $props();

	const queryClient = new QueryClient({
		defaultOptions: {
			queries: {
				staleTime: 10_000,
				refetchOnWindowFocus: false,
				retry: 1
			}
		}
	});

	const navLinks = [
		{ href: '/', label: 'Dashboard', icon: LayoutDashboard },
		{ href: '/tasks', label: 'Tareas', icon: ListTodo },
		{ href: '/definitions', label: 'Definiciones', icon: Code2 },
		{ href: '/schedules', label: 'Schedules', icon: CalendarClock },
		{ href: '/webhooks', label: 'Webhooks', icon: Webhook }
	];

	function isActive(href: string) {
		if (href === '/') return page.url.pathname === '/';
		return page.url.pathname.startsWith(href);
	}

	onMount(() => {
		const disconnect = connectEventSource(queryClient);
		return disconnect;
	});
</script>

<svelte:head>
	<title>Task Orchestrator</title>
</svelte:head>

<QueryClientProvider client={queryClient}>
	<div class="flex h-screen overflow-hidden bg-[#0e0e10] text-[#e7e4ec]">

		<!-- Sidebar: 48px colapsada, 200px en hover -->
		<aside
			class="group relative flex h-full w-12 shrink-0 flex-col border-r border-[#1f1f24]
				   bg-[#0e0e10] transition-all duration-200 ease-out hover:w-48 overflow-hidden z-50"
		>
			<!-- Logo -->
			<div class="flex h-12 items-center border-b border-[#1f1f24] px-3 shrink-0">
				<div class="flex h-6 w-6 shrink-0 items-center justify-center rounded bg-[#5516be]">
					<Zap size={14} class="text-white" />
				</div>
				<span class="ml-3 whitespace-nowrap text-xs font-semibold tracking-widest opacity-0 transition-opacity duration-150 group-hover:opacity-100 text-[#e7e4ec]">
					ORCHESTRATOR
				</span>
			</div>

			<!-- Nav links -->
			<nav class="flex flex-1 flex-col gap-0.5 p-2">
				{#each navLinks as link}
					{@const active = isActive(link.href)}
					<a
						href={link.href}
						class="flex h-8 items-center rounded-md px-2 text-sm transition-colors
							   {active
								? 'bg-[#5516be]/20 text-[#d0bcff]'
								: 'text-[#a09da1] hover:bg-[#1f1f24] hover:text-[#e7e4ec]'}"
					>
						<link.icon size={16} class="shrink-0 {active ? 'text-[#d0bcff]' : ''}" />
						<span class="ml-3 whitespace-nowrap opacity-0 transition-opacity duration-150 group-hover:opacity-100">
							{link.label}
						</span>
						{#if active}
							<span class="ml-auto h-1 w-1 shrink-0 rounded-full bg-[#d0bcff] opacity-0 group-hover:opacity-100"></span>
						{/if}
					</a>
				{/each}
			</nav>

			<!-- Settings bottom -->
			<div class="border-t border-[#1f1f24] p-2">
				<a
					href="/settings"
					class="flex h-8 items-center rounded-md px-2 text-sm text-[#a09da1] transition-colors hover:bg-[#1f1f24] hover:text-[#e7e4ec]"
				>
					<Settings size={16} class="shrink-0" />
					<span class="ml-3 whitespace-nowrap opacity-0 transition-opacity duration-150 group-hover:opacity-100">
						Ajustes
					</span>
				</a>
			</div>
		</aside>

		<!-- Contenido principal -->
		<main class="flex-1 overflow-auto">
			{@render children()}
		</main>

	</div>
</QueryClientProvider>
