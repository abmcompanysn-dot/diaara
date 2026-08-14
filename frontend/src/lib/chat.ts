'use client';

import {
  collection,
  doc,
  onSnapshot,
  orderBy,
  query,
  serverTimestamp,
  setDoc,
  addDoc,
  where,
  type Unsubscribe,
} from 'firebase/firestore';
import { firestoreDb, ensureFirestoreIdentity } from '@/lib/firebase';

export interface ChatMessage {
  id: string;
  senderId: string;
  senderRole: 'buyer' | 'vendor';
  body: string;
  createdAt: Date | null;
}

// Une conversation par binôme client-vendeur (tous produits confondus) —
// conversationId déterministe, pas besoin de le stocker ailleurs.
export function conversationId(buyerId: string, vendorId: string): string {
  return `${buyerId}_${vendorId}`;
}

export function listenToConversation(
  buyerId: string,
  vendorId: string,
  onMessages: (messages: ChatMessage[]) => void,
  onError?: (error: Error) => void
): Unsubscribe {
  const db = firestoreDb();
  const messagesRef = collection(db, 'conversations', conversationId(buyerId, vendorId), 'messages');
  const q = query(messagesRef, orderBy('createdAt', 'asc'));
  return onSnapshot(q, (snapshot) => {
    onMessages(
      snapshot.docs.map((d) => ({
        id: d.id,
        senderId: d.data().senderId,
        senderRole: d.data().senderRole,
        body: d.data().body,
        createdAt: d.data().createdAt?.toDate?.() || null,
      }))
    );
  }, onError);
}

export async function sendMessage(
  buyerId: string,
  vendorId: string,
  buyerName: string,
  vendorName: string,
  senderId: string,
  senderRole: 'buyer' | 'vendor',
  body: string
): Promise<void> {
  await ensureFirestoreIdentity();
  const db = firestoreDb();
  const convId = conversationId(buyerId, vendorId);

  // setDoc merge : crée la conversation si elle n'existe pas encore, sans
  // écraser l'historique si elle existe déjà.
  await setDoc(
    doc(db, 'conversations', convId),
    { buyerId, vendorId, buyerName, vendorName, updatedAt: serverTimestamp(), lastMessage: body },
    { merge: true }
  );

  await addDoc(collection(db, 'conversations', convId, 'messages'), {
    senderId,
    senderRole,
    body,
    createdAt: serverTimestamp(),
  });
}

export interface ConversationSummary {
  id: string;
  buyerId: string;
  vendorId: string;
  buyerName: string;
  vendorName: string;
  lastMessage: string;
}

// listenToVendorInbox : toutes les conversations d'un vendeur, pour son écran
// de messagerie (frontend/src/app/(app)/vendor/messages).
export function listenToVendorInbox(
  vendorId: string,
  onConversations: (list: ConversationSummary[]) => void,
  onError?: (error: Error) => void
): Unsubscribe {
  const db = firestoreDb();
  const q = query(collection(db, 'conversations'), where('vendorId', '==', vendorId));
  return onSnapshot(q, (snapshot) => {
    onConversations(
      snapshot.docs.map((d) => ({
        id: d.id,
        buyerId: d.data().buyerId,
        vendorId: d.data().vendorId,
        buyerName: d.data().buyerName,
        vendorName: d.data().vendorName,
        lastMessage: d.data().lastMessage || '',
      }))
    );
  }, onError);
}
