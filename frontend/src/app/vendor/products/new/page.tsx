'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';

export default function NewProductPage() {
  const router = useRouter();
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [price, setPrice] = useState('');
  const [category, setCategory] = useState('ebook');
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (!file) {
      setError('Veuillez sélectionner un fichier');
      return;
    }
    const priceNum = parseInt(price, 10);
    if (isNaN(priceNum) || priceNum < 0) {
      setError('Prix invalide');
      return;
    }

    setUploading(true);
    try {
      // Étape 1 : upload du fichier
      const uploadForm = new FormData();
      uploadForm.append('file', file);
      const uploadResult = await api.uploadFile(uploadForm);

      // Étape 2 : création du produit
      const productForm = new FormData();
      productForm.append('title', title);
      productForm.append('description', description);
      productForm.append('price_cfa', String(priceNum));
      productForm.append('category', category);
      productForm.append('file_key', uploadResult.file_key);

      await api.createProduct(productForm);
      router.push('/vendor/products');
    } catch (err: any) {
      setError(err.message || 'Échec de la création du produit');
    } finally {
      setUploading(false);
    }
  };

  return (
    <main className="min-h-screen p-8 max-w-2xl mx-auto">
      <nav className="mb-6">
        <Link href="/vendor/products" className="text-green-600">← Mes produits</Link>
      </nav>

      <h1 className="text-3xl font-bold mb-6">Nouveau produit</h1>

      {error && <div className="mb-4 p-3 bg-red-50 text-red-600 rounded">{error}</div>}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium mb-1">Titre *</label>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="w-full p-2 border rounded"
            required
          />
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">Description</label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="w-full p-2 border rounded min-h-32"
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium mb-1">Prix (FCFA) *</label>
            <input
              type="number"
              value={price}
              onChange={(e) => setPrice(e.target.value)}
              className="w-full p-2 border rounded"
              required
              min={0}
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">Catégorie *</label>
            <select
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              className="w-full p-2 border rounded"
            >
              <option value="subscription">Clé d'abonnement</option>
              <option value="account">Compte</option>
              <option value="ebook">Ebook</option>
              <option value="pdf">PDF</option>
              <option value="other">Autre</option>
            </select>
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">Fichier à vendre *</label>
          <input
            type="file"
            onChange={(e) => setFile(e.target.files?.[0] || null)}
            className="w-full p-2 border rounded"
            required
          />
          <p className="mt-1 text-xs text-green-900/50">Taille max : 50 Mo</p>
        </div>

        <button
          type="submit"
          disabled={uploading}
          className="w-full py-3 gradient-green text-white rounded-lg font-semibold hover:opacity-95 disabled:opacity-50"
        >
          {uploading ? 'Publication en cours...' : 'Publier le produit'}
        </button>
      </form>
    </main>
  );
}
