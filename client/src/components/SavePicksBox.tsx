interface SavePicksBoxProps {
	isSaving: boolean;
	onSave: () => void;
	onRevert: () => void;
}

function SavePicksBox({ isSaving, onSave, onRevert }: SavePicksBoxProps) {
	return (
		<div className="border-4 border-black flex flex-col p-2">
			<p>Unsaved event picks</p>
			{isSaving ? (
				<p className="text-center">Saving...</p>
			) : (
				<div className="flex flex-row justify-evenly mt-2">
					<button
						className="bg-green-500 hover:bg-green-600 p-2 rounded-md"
						onClick={onSave}
						type="button"
					>
						Save
					</button>
					<button
						className="bg-red-500 hover:bg-red-600 p-2 rounded-md"
						onClick={onRevert}
						type="button"
					>
						Revert
					</button>
				</div>
			)}
		</div>
	);
}

export default SavePicksBox;
