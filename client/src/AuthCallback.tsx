import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { authCallback } from "./api";
import FullscreenText from "./components/FullscreenText";

function AuthCallback() {
  const navigate = useNavigate();

  useEffect(() => {
    authCallback().then(() => {
      navigate("/");
    });
  }, [navigate]);

  return <FullscreenText text="Authenticating..." />;
}

export default AuthCallback;
