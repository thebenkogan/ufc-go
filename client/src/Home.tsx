import { useState } from "react";
import { postPicks, useEvent, usePicks, useUser } from "./api";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import FullscreenText from "./components/FullscreenText";
import SavePicksBox from "./components/SavePicksBox";
import EventDisplay from "./components/EventDisplay";
import toast from "react-hot-toast";
import type { Event } from "./types";

function Home() {
	const eventId =
		new URLSearchParams(window.location.search).get("id") ?? "latest";
	const { data: event } = useEvent(eventId);
	const user = useUser();

	if (!event || user.isLoading) {
		return <FullscreenText text="Loading..." />;
	}

	return user.data ? (
		<EventWithPickControl eventId={eventId} event={event} />
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
	eventId: string;
	event: Event;
}

function EventWithPickControl({ event, eventId }: EventWithPickControlProps) {
	const picks = usePicks(eventId);
	const eventPicks = picks.data?.winners;
	const [localPicks, setLocalPicks] = useState<string[]>([]);
	const [prevServerPicks, setPrevServerPicks] = useState(eventPicks);
	const queryClient = useQueryClient();

	const picksMutation = useMutation({
		mutationFn: (picks: string[]) => postPicks(eventId, picks),
		onSuccess: () => {
			return queryClient.invalidateQueries({
				queryKey: [`events/${eventId}/picks`],
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
			toast.error("You can't pick both fighters in a fight");
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
