// SPDX-License-Identifier: GPL-3.0-or-later
package io.myadblock;
import android.content.Context;
import android.net.*;
import org.json.*;
import java.net.InetAddress;
import java.util.*;
/** app_process helper. Reads real LinkProperties; never parses dumpsys/net.dns properties. */
public final class NetworkProbe {
 public static void main(String[] args) {
  try {
   if(android.os.Looper.getMainLooper()==null) android.os.Looper.prepareMainLooper();
   Class<?> at=Class.forName("android.app.ActivityThread");
   Object thread=at.getMethod("systemMain").invoke(null);
   Context ctx=(Context)at.getMethod("getSystemContext").invoke(thread);
   ConnectivityManager cm=(ConnectivityManager)ctx.getSystemService(Context.CONNECTIVITY_SERVICE);
   Network active=cm.getActiveNetwork();
   NetworkCapabilities ac=active==null?null:cm.getNetworkCapabilities(active);
   boolean vpn=ac!=null&&ac.hasTransport(NetworkCapabilities.TRANSPORT_VPN);
   List<Network> candidates=new ArrayList<>();
   for(Network n:cm.getAllNetworks()) {
    NetworkCapabilities c=cm.getNetworkCapabilities(n);
    if(c!=null&&c.hasTransport(NetworkCapabilities.TRANSPORT_VPN)){vpn=true;continue;}
    if(c!=null&&c.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)) candidates.add(n);
   }
   Network selected=null;
   if(!vpn&&active!=null&&candidates.contains(active)) selected=active;
   else if(candidates.size()==1) selected=candidates.get(0);
   JSONObject o=new JSONObject();o.put("vpn",vpn);o.put("reliable",selected!=null);
   JSONArray dns=new JSONArray();String iface="";
   if(selected!=null){LinkProperties p=cm.getLinkProperties(selected);if(p!=null){iface=p.getInterfaceName();for(InetAddress a:p.getDnsServers()) dns.put(a.getHostAddress());}else{o.put("reliable",false);}}
   o.put("interface",iface);o.put("dns",dns);System.out.println(o.toString());
  } catch(Throwable t) { System.out.println("{\"reliable\":false,\"vpn\":false,\"dns\":[],\"interface\":\"\"}");System.exit(1); }
 }
}
