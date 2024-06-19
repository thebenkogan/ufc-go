import { useState } from "react";
import { postPicks, useEvent, useEventPicks } from "./api";
import { useMutation, useQueryClient } from "@tanstack/react-query";

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
    return <div>Error: {error.message}</div>;
  }

  if (!latestEvent) {
    return (
      <div className="flex h-screen justify-center items-center text-xl">
        Loading...
      </div>
    );
  }

  const clickFighterHandler = (fighter: string, opponent: string) => {
    return () => {
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
  };

  const startTime = new Date(latestEvent.start_time);
  const hasPickChanges =
    localPicks.sort().toString() !==
    (latestPicks.data?.winners.sort().toString() ?? "");

  return (
    <>
      {hasPickChanges && (
        <div className="absolute right-10 top-1/2 -translate-y-1/2 p-2 border-4 border-black flex flex-col">
          <p>Unsaved event picks</p>
          {picksMutation.isPending ? (
            <p className="text-center">Saving...</p>
          ) : (
            <div className="flex flex-row justify-evenly mt-2">
              <button
                className="bg-green-500 hover:bg-green-600 p-2 rounded-md"
                onClick={() => {
                  picksMutation.mutate(localPicks);
                }}
              >
                Save
              </button>
              <button
                className="bg-red-500 hover:bg-red-600 p-2 rounded-md"
                onClick={() => {
                  setLocalPicks(latestPicks.data?.winners || []);
                }}
              >
                Revert
              </button>
            </div>
          )}
        </div>
      )}
      <div className="flex flex-col h-screen items-stretch">
        <div>
          <h1 className="text-4xl text-center">Event {latestEvent.id}</h1>
          <h2 className="text-xl text-center">
            {startTime.toDateString()} at {startTime.toLocaleTimeString()}
          </h2>
        </div>
        <div className="flex-grow flex flex-col justify-center items-center">
          {latestEvent.fights.map((fight, index) => (
            <div
              key={index}
              className="flex items-center space-x-2 py-1 my-0.5 w-7/12 border-4 border-black bg-slate-400"
            >
              <div className="flex justify-evenly w-full gap-2 items-center px-4">
                <button
                  onClick={clickFighterHandler(
                    fight.fighters[0],
                    fight.fighters[1]
                  )}
                  className={`flex-1 ${
                    localPicks.includes(fight.fighters[0])
                      ? "bg-green-500 hover:bg-green-600"
                      : "bg-slate-500 hover:bg-slate-600"
                  } p-2 rounded-lg font-bold`}
                >
                  {fight.fighters[0]}
                </button>
                <p className="flex-1 text-center text-xl font-bold">vs</p>
                <button
                  onClick={clickFighterHandler(
                    fight.fighters[1],
                    fight.fighters[0]
                  )}
                  className={`flex-1 ${
                    localPicks.includes(fight.fighters[1])
                      ? "bg-green-500 hover:bg-green-600"
                      : "bg-slate-500 hover:bg-slate-600"
                  } p-2 rounded-lg font-bold`}
                >
                  {fight.fighters[1]}
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </>
  );
}

export default Home;
