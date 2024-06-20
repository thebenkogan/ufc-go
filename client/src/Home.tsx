import { useState } from "react";
import { postPicks, useEventWithPicks } from "./api";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import FullscreenText from "./components/FullscreenText";
import SavePicksBox from "./components/SavePicksBox";
import EventDisplay from "./components/EventDisplay";
import toast from "react-hot-toast";

const HOME_EVENT_ID = "latest";

function Home() {
  const {
    data: eventData,
    error,
    isLoading,
  } = useEventWithPicks(HOME_EVENT_ID);
  const event = eventData?.event;
  const eventPicks = eventData?.winners;
  console.log(eventData);
  const [localPicks, setLocalPicks] = useState<string[]>([]);
  const [prevServerPicks, setPrevServerPicks] = useState(eventPicks);
  const queryClient = useQueryClient();

  const picksMutation = useMutation({
    mutationFn: (picks: string[]) => postPicks(HOME_EVENT_ID, picks),
    onSuccess: () => {
      return queryClient.invalidateQueries({
        queryKey: ["events/latest/picks"],
      });
    },
  });

  if (prevServerPicks !== eventPicks) {
    setPrevServerPicks(eventPicks);
    if (localPicks.length === 0) {
      setLocalPicks(eventPicks || []);
    }
  }

  if (error) {
    return <FullscreenText text="Error loading event" />;
  }

  if (!event) {
    return <FullscreenText text="Loading event..." />;
  }

  const eventHasStarted =
    event.start_time === "LIVE" || new Date() > new Date(event.start_time);

  const clickFighterHandler = (fighter: string, opponent: string) => {
    if (isLoading) {
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
          score={eventData?.score}
        />
      </div>
    </>
  );
}

export default Home;
