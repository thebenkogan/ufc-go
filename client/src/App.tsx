import { useEffect, useState } from "react";
import { get } from "./api";
import { Route, Routes, useNavigate } from "react-router-dom";
import { Event } from "./types";

function App() {
  return (
    <Routes>
      <Route path="/" element={<Home />} />
      <Route path="/auth/google/callback" element={<AuthCallback/>} />
    </Routes>
  );
}

function Home() {
  const [data, setData] = useState<Event>();

  useEffect(() => {
    get<Event>("events/latest").then((res) => {
      setData(res);
    });
  }, []);

  return <pre>{JSON.stringify(data, null, 2)}</pre>;
}

function AuthCallback() {
  const navigate = useNavigate();
  const path = "auth/google/callback" + window.location.search;
  
  useEffect(() => {
    get(path).then(() => {
        navigate("/");
    });
  }, [path, navigate]);

  return <div className="bg-blue-300">auth callback</div>;
}

export default App;
