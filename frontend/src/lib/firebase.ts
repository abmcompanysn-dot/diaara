'use client';

import { initializeApp, getApps, type FirebaseApp } from 'firebase/app';
import { getAuth, GoogleAuthProvider, signInWithPopup, signInAnonymously, type Auth } from 'firebase/auth';
import { getFirestore, type Firestore } from 'firebase/firestore';

const firebaseConfig = {
  apiKey: process.env.NEXT_PUBLIC_FIREBASE_API_KEY,
  authDomain: process.env.NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN,
  projectId: process.env.NEXT_PUBLIC_FIREBASE_PROJECT_ID,
  appId: process.env.NEXT_PUBLIC_FIREBASE_APP_ID,
};

// firebaseEnabled : tant que le projet Firebase n'est pas configuré (voir
// .env.example), on désactive proprement le bouton Google et le chat plutôt
// que de planter au chargement de la page.
export const firebaseEnabled = Boolean(firebaseConfig.apiKey && firebaseConfig.projectId && firebaseConfig.appId);

let app: FirebaseApp | undefined;
let authInstance: Auth | undefined;
let dbInstance: Firestore | undefined;

function getFirebaseApp(): FirebaseApp {
  if (!app) {
    app = getApps().length ? getApps()[0] : initializeApp(firebaseConfig as Record<string, string>);
  }
  return app;
}

export function firebaseAuth(): Auth {
  if (!authInstance) authInstance = getAuth(getFirebaseApp());
  return authInstance;
}

export function firestoreDb(): Firestore {
  if (!dbInstance) dbInstance = getFirestore(getFirebaseApp());
  return dbInstance;
}

// signInWithGoogle : ouvre la popup Google, retourne l'ID token Firebase à
// envoyer au backend (POST /api/auth/google) — Firebase ne sert que de
// vérificateur d'identité, DIARRA garde son propre système de session.
export async function signInWithGoogle(): Promise<string> {
  const provider = new GoogleAuthProvider();
  const result = await signInWithPopup(firebaseAuth(), provider);
  return result.user.getIdToken();
}

// ensureFirestoreIdentity : pour le chat (Firestore), un utilisateur DIARRA
// qui ne s'est jamais connecté via Google a quand même besoin d'un UID
// Firebase pour que les règles de sécurité Firestore puissent l'identifier —
// une connexion anonyme suffit, elle n'est utilisée que côté Firestore.
export async function ensureFirestoreIdentity(): Promise<string> {
  const auth = firebaseAuth();
  if (auth.currentUser) return auth.currentUser.uid;
  const result = await signInAnonymously(auth);
  return result.user.uid;
}
