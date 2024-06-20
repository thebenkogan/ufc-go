import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { authCallback } from "./api";

function AuthCallback() {
  const navigate = useNavigate();

  useEffect(() => {
    authCallback().then(() => {
      navigate("/");
    });
  }, [navigate]);

  return (
    <div className="flex h-screen justify-center items-center text-xl">
      Loading...
    </div>
  );
}

export default AuthCallback;
