// src/remote-config.js
export const getConfig = async (name) => {
    // Appel vers votre microservice Go
    const response = await fetch(`/${name}/api/v1/mfe-setup`); 
    const data = await response.json();
    let metadata={}
    if (data && Object.keys(data).length > 0) {
        const key=Object.keys(data)[0];
        if(data[key].error){
            throw new Error(data[key].error);
        }
        metadata['icon'] = data[key]['icon']
        metadata['name'] = data[key]['name']
    }
    
    // On retourne l'objet au format que le Shell attend
    return {
        metadata: metadata,
        menu: data
    };
};