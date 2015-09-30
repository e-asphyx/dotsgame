window.Utils = window.Utils || {};

(function() {
	Utils.modalToggle = function(selector, opened) {
		$(selector).toggleClass("open", opened);
	};

	Utils.RGB = function(v) {
		return "rgb(" + v[0] + "," + v[1] + "," + v[2] + ")";
	};

	Utils.RGBA = function(v, a) {
		return "rgba(" + v[0] + "," + v[1] + "," + v[2] + "," + a + ")";
	};

	Utils.Proto = function(obj, elem, val) {
		if(!obj[elem]) {
			obj.constructor.prototype[elem] = val;
		}
	};

	$(document).ready(function() {
		/* Dropdown helper */
		$("body").on("click", ".dropdown", function(evt) {
			var container = $(".dropdown-container");

			if(!container.is(evt.target) && container.has(evt.target).length === 0) {
				evt.preventDefault();

				var opened = $(evt.currentTarget).hasClass("open");
				$(".dropdown").removeClass("open"); /* close others */
				$(evt.currentTarget).toggleClass("open", !opened);
			}
		});

		$("body").click(function(evt) {
			var dropdown = $(".dropdown");

			if(!dropdown.is(evt.target) && dropdown.has(evt.target).length === 0) {
				dropdown.removeClass("open");
			}
		});

		/* Modal helper */
		$("body").on("click", "a.modal-close", function(evt) {
			console.log(evt);
			evt.preventDefault();
			$(evt.target).closest(".modal").removeClass("open");
		});

		$("body").on("click", ".modal", function(evt) {
			console.log(evt);
			var body = $(".modal-content");

			if(!body.is(evt.target) && body.has(evt.target).length === 0) {
				$(evt.target).closest(".modal").removeClass("open");
			}
		});
	});
})();
