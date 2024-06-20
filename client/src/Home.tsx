import { useState } from "react";
import { postPicks, useEvent, useEventPicks } from "./api";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import FullscreenText from "./components/FullscreenText";
import SavePicksBox from "./components/SavePicksBox";
import EventDisplay from "./components/Event";

function Home() {
  const { data: latestEvent, error } = useEvent("latest");
  const latestPicks = useEventPicks("latest");
  const [localPicks, setLocalPicks] = useState<string[]>([]);
  const [prevServerPicks, setPrevServerPicks] = useState(latestPicks.data);
  const queryClient = useQueryClient();

  const picksMutation = useMutation({
    mutationFn: (picks: string[]) => postPicks("latest", picks),
    onSuccess: () => {
      return queryClient.invalidateQueries({
        queryKey: ["events/latest/picks"],
      });
    },
  });

  if (prevServerPicks !== latestPicks.data) {
    setPrevServerPicks(latestPicks.data);
    if (localPicks.length === 0) {
      setLocalPicks(latestPicks.data?.winners || []);
    }
  }

  if (error) {
    return <FullscreenText text="Error loading event" />;
  }

  if (!latestEvent) {
    return <FullscreenText text="Loading event..." />;
  }

  const clickFighterHandler = (fighter: string, opponent: string) => {
    if (latestPicks.isLoading) {
      return;
    }
    if (localPicks.includes(fighter)) {
      setLocalPicks(localPicks.filter((pick) => pick !== fighter));
    } else if (localPicks.includes(opponent)) {
      alert("You can't pick both fighters in a fight");
    } else {
      setLocalPicks([...localPicks, fighter]);
    }
  };

  const hasPickChanges =
    localPicks.sort().toString() !==
    (latestPicks.data?.winners.sort().toString() ?? "");

  return (
    <>
      {hasPickChanges && (
        <div className="absolute right-10 top-1/2 -translate-y-1/2">
          <SavePicksBox
            isSaving={picksMutation.isPending}
            onSave={() => picksMutation.mutate(localPicks)}
            onRevert={() => setLocalPicks(latestPicks.data?.winners || [])}
          />
        </div>
      )}
      <div className="h-screen">
        <EventDisplay
          event={latestEvent}
          picks={localPicks}
          onClickFighter={clickFighterHandler}
        />
      </div>
    </>
  );
}

export default Home;
