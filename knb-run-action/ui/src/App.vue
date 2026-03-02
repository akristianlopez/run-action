<template>
  <div class="bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden">
    <div class="bg-slate-50 border-b border-slate-200 p-4">
      <h3 class="text-lg font-bold text-slate-800">Gestion de l'Organisation</h3>
      <p class="text-sm text-slate-500">Vue d'ensemble et membres</p>
    </div>
    
    <div class="p-6">
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
        <div class="p-4 bg-indigo-50 rounded-lg border border-indigo-100">
          <span class="block text-xs font-semibold text-indigo-600 uppercase">Membres</span>
          <span class="text-2xl font-bold text-indigo-900">24</span>
        </div>
        <div class="p-4 bg-emerald-50 rounded-lg border border-emerald-100">
          <span class="block text-xs font-semibold text-emerald-600 uppercase">Statut</span>
          <span class="text-2xl font-bold text-emerald-900">Actif</span>
        </div>
        <div class="p-4 bg-amber-50 rounded-lg border border-amber-100">
          <span class="block text-xs font-semibold text-amber-600 uppercase">Projets</span>
          <span class="text-2xl font-bold text-amber-900">12</span>
        </div>
      </div>

      <table class="w-full text-left">
        <thead>
          <tr class="text-slate-400 text-sm uppercase">
            <th class="pb-3 font-medium">Nom</th>
            <th class="pb-3 font-medium">Rôle</th>
            <th class="pb-3 font-medium">Dernière activité</th>
          </tr>
        </thead>
        <tbody class="text-slate-600 divide-y divide-slate-100">
          <tr>
            <td class="py-3 font-medium text-slate-900">Jean Dupont</td>
            <td class="py-3">Administrateur</td>
            <td class="py-3 text-sm">Il y a 2h</td>
          </tr>
          <tr>
            <td class="py-3 font-medium text-slate-900">Marie Curie</td>
            <td class="py-3">Éditeur</td>
            <td class="py-3 text-sm">Hier</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script>
  export default {
    name: 'OrganizationDashboard',
    setup() {
      return {};
    }
  }
</script>