powersupplylabels = { 0: "Alimentazione OK", 1: "In blackout", 2: "Adiacente ad un blackout" };

oldsize = "";
oldpadding = "";
target = undefined;

function isCompanyVisible(companyLabelObj) {
	return !companyLabelObj.hasClass("company-deselected");
}

function showCompany(companyLabelObj) {
	var cid = companyLabelObj.data("company-id");

	companyLabelObj.removeClass("company-deselected");
	$(".realnode[data-owner-id=" + cid + "]").removeClass("company-deselected");
}

function hideCompany(companyLabelObj) {
	var cid = companyLabelObj.data("company-id");

	companyLabelObj.addClass("company-deselected");
	$(".realnode[data-owner-id=" + cid + "]").addClass("company-deselected");
}

$(function() {
	$('#modal-node').on('hidden.bs.modal', function() {
		target.css("font-size", oldsize);
		target.css("padding", oldpadding);
	});

	$(".company-label").click(function(e) {
		var target = $(this);
		var wasHidden = !isCompanyVisible(target);

		if (wasHidden) {
			showCompany(target);
		} else {
			hideCompany(target);
		}
	});

	$("#company-selectall").click(function(e) {
		$(".company-label").each(function() {
			showCompany($(this));
		});
	});

	$("#company-selectmine").click(function(e) {
		$(".company-label").each(function() {
			hideCompany($(this));
		});
		$(".company-label.company-mine").each(function() {
			showCompany($(this));
		});
	});

	$("#company-hideall").click(function(e) {
		$(".company-label").each(function() {
			hideCompany($(this));
		});
	});

	$("td:not(.fakenode)").click(function(e) {
		target = $(this);
		var oldlink = $("#sel-owner-link").attr("href");

		if (target.data("owner-name") != undefined) {
			$("#sel-owner-name").text(target.data("owner-name"));
		} else {
			$("#sel-owner-name").text("");
		}
		$("#sel-owner-link").attr("href", oldlink.replace(/[^\/]*$/, target.data("owner-id")));
		$("#sel-x").text(target.data("x"));
		$(".sel-x").val(target.data("x"));
		$("#sel-y").text(target.data("y"));
		$(".sel-y").val(target.data("y"));

		var owner_id = target.data("owner-id");
		if (owner_id != undefined) {
			$(".sel-owner-id").val(owner_id);
			$("#sel-mainaction").text("Noleggia");
		} else {
			$("#sel-mainaction").text("Compra");
		}

		$("#sel-yield").text(target.data("yield") / 100);
		$("#sel-blackoutprob").text(Math.round(target.data("blackoutp") * 10000) / 100);
		$("#sel-stability").text(target.data("stability") + 1);
		$("#sel-buycost").text(target.data("buycost") / 100);
		if (target.data("investcost") != undefined) {
			$("#investbutton").show();
			$("#sel-investcost").text(target.data("investcost") / 100);
		} else {
			$("#investbutton").hide();
		}
		$("#sel-newyield").text(target.data("newyield") / 100);

		powersupply = target.data("powersupply");
		if (powersupply != 0) {
			$("#sel-powersupply-p").show();
			$("#sel-powersupply").text(powersupplylabels[powersupply]);
		} else {
			$("#sel-powersupply-p").hide();
		}

		tenantslist = target.data("tenants").trim();
		if (tenantslist != "") {
			$("#sel-tenants-p").show();
			tenants = tenantslist.split("|");
			tenants = $.map(tenants, function(value, index) {
				return value.trim();
			});
			tenants = $.grep(tenants, function(value, index) {
				return value != "";
			});
			$("#sel-tenants").text(tenants.join(", "));
		} else {
			$("#sel-tenants-p").hide();
		}

		oldsize = target.css("font-size");
		oldpadding = target.css("padding");
		target.css("font-size", "x-large");
		target.css("padding", "10px");
	});
});
