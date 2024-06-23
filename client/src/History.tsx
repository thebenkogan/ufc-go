import { useAllPicks } from "./api";
import FullscreenText from "./components/FullscreenText";
import PicksTable from "./components/PicksTable";

function History() {
	const { data: picksHistory } = useAllPicks();

	if (!picksHistory) {
		return <FullscreenText text="Loading..." />;
	}

	return (
		<div className="flex flex-col bg-red items-center h-screen">
			<h1 className="text-xl font-bold my-2">Your Past Picks</h1>
			<PicksTable picks={picksHistory} />
		</div>
	);
}

export default History;
