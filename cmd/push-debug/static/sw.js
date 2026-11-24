self.addEventListener('push', event => {
 event.waitUntil((async () => {
  let payload={};try{payload=event.data?.json()||{};}catch{/* Show a visible fallback. */}
  await self.registration.showNotification(typeof payload.title==='string'?payload.title:'Холодильник',{
   body:typeof payload.body==='string'?payload.body:'Проверьте сроки годности.',
   tag:typeof payload.tag==='string'?payload.tag:'fridge-daily',data:{url:'/'},
  });
 })());
});
self.addEventListener('notificationclick',event=>{
 event.notification.close();
 event.waitUntil((async()=>{
  const windows=await self.clients.matchAll({type:'window',includeUncontrolled:true});
  if(windows.length){await windows[0].focus();return;}
  await self.clients.openWindow('/');
 })());
});
