var i;
var tds;

tds = document.getElementsByTagName("td");
Xs = document.getElementsByName("X");
Ys = document.getElementsByName("Y");
buyCost = document.getElementById("buyCost");
investCost = document.getElementById("investCost");

for (i = 0; i < tds.length; i++) {
	tds[i].onclick = function(e) {
		var x = e.target.dataset.x;
		var y = e.target.dataset.y;
		Xs[0].value = x;
		Xs[1].value = x;
		Ys[0].value = y;
		Ys[1].value = y;

		var xhttp = new XMLHttpRequest();
		xhttp.onreadystatechange = function() {
			if (this.readyState == 4 && this.status == 200) {
				var costs = JSON.parse(this.responseText);

				buyCost.innerHTML = costs.BuyCost / 100
				investCost.innerHTML = costs.InvestCost / 100
			}
		};

		xhttp.open("GET", "/game/map/costs/" + x + "/" + y, true);
		xhttp.send();
	}   
}
