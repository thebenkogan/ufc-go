import { useState } from "react";
import { postPicks, useEvent, usePicks, useUser } from "./api";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import FullscreenText from "./components/FullscreenText";
import SavePicksBox from "./components/SavePicksBox";
import EventDisplay from "./components/EventDisplay";
import toast from "react-hot-toast";
import type { Event } from "./types";
import { useSearchParams } from "react-router-dom";

function Home() {
	const [searchParams] = useSearchParams();
	const eventId = searchParams.get("id") ?? "latest";
	const { data: event } = useEvent(eventId);
	const user = useUser();

	if (!event || user.isLoading) {
		return <FullscreenText text="Loading..." />;
	}

	return user.data ? (
		<EventWithPickControl event={event} />
	) : (
		<div className="h-screen">
			<EventDisplay
				event={event}
				picks={[]}
				onClickFighter={() => {}}
				score={undefined}
			/>
		</div>
	);
}

interface EventWithPickControlProps {
	event: Event;
}

function EventWithPickControl({ event }: EventWithPickControlProps) {
	const picks = usePicks(event.id);
	const eventPicks = picks.data?.winners;
	const [localPicks, setLocalPicks] = useState<string[]>(eventPicks || []);
	const [prevServerPicks, setPrevServerPicks] = useState(eventPicks);
	const queryClient = useQueryClient();

	const picksMutation = useMutation({
		mutationFn: (picks: string[]) => postPicks(event.id, picks),
		onSuccess: () => {
			return queryClient.invalidateQueries({
				queryKey: [`events/${event.id}/picks`],
			});
		},
	});

	if (prevServerPicks !== eventPicks) {
		setPrevServerPicks(eventPicks);
		if (localPicks.length === 0) {
			setLocalPicks(eventPicks || []);
		}
	}

	const eventHasStarted =
		event.start_time === "LIVE" || new Date() > new Date(event.start_time);

	const clickFighterHandler = (fighter: string, opponent: string) => {
		if (picks.isLoading) {
			toast.loading("Loading your picks...");
			return;
		}
		if (eventHasStarted) {
			toast.error("Event has already started, picks are locked");
			return;
		}
		if (localPicks.includes(fighter)) {
			setLocalPicks(localPicks.filter((pick) => pick !== fighter));
		} else if (localPicks.includes(opponent)) {
			setLocalPicks(
				[...localPicks, fighter].filter((pick) => pick !== opponent),
			);
		} else {
			setLocalPicks([...localPicks, fighter]);
		}
	};

	const hasPickChanges =
		localPicks.sort().toString() !== (eventPicks?.sort().toString() ?? "");

	return (
		<>
			{hasPickChanges && (
				<div className="absolute right-10 top-1/2 -translate-y-1/2">
					<SavePicksBox
						isSaving={picksMutation.isPending}
						onSave={() => picksMutation.mutate(localPicks)}
						onRevert={() => setLocalPicks(eventPicks || [])}
					/>
				</div>
			)}
			<div className="h-screen">
				<EventDisplay
					event={event}
					picks={localPicks}
					onClickFighter={clickFighterHandler}
					score={picks.data?.score}
				/>
			</div>
		</>
	);
}

export default Home;
