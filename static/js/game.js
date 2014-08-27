var game = {};
(function() {
	game.style = {
		board: {
			padding: 7.0,
			grid: {
				"strokeStyle": "#cccccc",
 				"lineWidth": 1.0
			},
		},

		playerA: {
			radius: 6.5,
			point: {
				"fillStyle": "#00ff00"
			},
			area: {
				"fillStyle": "rgba(0,255,0,0.2)",
 				"strokeStyle": "#00ff00",
    			"lineCap": "round",
 				"lineJoin": "round",
 				"lineWidth": 1.0
			}
		},

		playerB: {
			radius: 6.5,
			point: {
				"fillStyle": "#ff0000"
			},
			area: {
				"fillStyle": "rgba(255,0,0,0.2)",
 				"strokeStyle": "#ff0000",
    			"lineCap": "round",
 				"lineJoin": "round",
 				"lineWidth": 1.0
			}
		}
	};

	CanvasRenderingContext2D.prototype.setStyle = function(obj) {
		_.extend(this, obj);
	};

	function RubberBand(objdata) {
		if(objdata instanceof Array) {
			/* Generic consructor */
			this.list = [this];
			this.val = objdata[0];
			this.list.push.apply(this.list, _.map(objdata.slice(1), function(obj) {return this._factory(obj);}, this));
		} else {
			this.list = objdata.list;
			this.val = objdata.val;
		}
	}

	RubberBand.prototype = {
		_idx: function() {
			return this.list.indexOf(this);
		},

		_factory: function(obj) {
			return new RubberBand({
				list: this.list,
				val: obj
			});
		},

		size: function() {
			return this.list.length;
		},
		
		next: function() {
			var idx = this._idx();
			return this.list[idx < this.list.length - 1 ? idx + 1 : 0];
		},
		
		prev: function() {
			var idx = this._idx();
			return this.list[idx > 0 ? idx - 1 : this.list.length - 1];
		},

		after: function(obj) {
			var idx = this._idx();
			var newObj = this._factory(obj);
			this.list.splice(idx + 1, 0, newObj);
			return newObj;
		},
		
		before: function(obj) {
			var idx = this._idx();
			var newObj = this._factory(obj);
			this.list.splice(idx, 0, newObj);
			return newObj;
		},

		remove: function() {
			var idx = this._idx();
			this.list.splice(idx, 1);
			if(this.list.length) {
				return this.list[idx < this.list.length ? idx : 0]
			}
		},

		getData: function() {
			return _.pluck(this.list, 'val');
		},

		find: function(predicate, ctx) {
			var args = Array.prototype.slice.call(arguments);
			args.unshift(this.list);
			return _.find.apply(_, args);
		},
		
		each: function(callback, ctx) {
			var args = Array.prototype.slice.call(arguments);
			args.unshift(this.list);
			return _.each.apply(_, args);
		},

		getAll: function(callback, ctx) {
			return this.list.slice(0);
		},

		deepClone: function() {
			var list = [];
			list.push.apply(list,
					_.map(this.list, function(obj) {
						return new RubberBand({
							list: list,
							val: obj.val
						});
					}));

			/* return valid link to copy of self */
			return list[this._idx()];
		}
	}

	function VacuumWrap(points) {
		this.points = points;
		
		var minx = this.minx = _.min(_.pluck(this.points, 'x'));
		var maxx = this.maxx = _.max(_.pluck(this.points, 'x'));
		var miny = this.miny = _.min(_.pluck(this.points, 'y'));
		var maxy = this.maxy = _.max(_.pluck(this.points, 'y'));
		
		this.bands = [];
		if(minx != maxx && miny != maxy) {
			/* make board map */
			this.map = [];
			_.times(maxy - miny + 1, function(n){this.map[n] = []}, this);
			_.each(this.points, function(p){this.map[p.y - miny][p.x - minx] = true;}, this);
			
			/* find spans */
			var spany = miny;
			var y = miny;
			while(y <= maxy) {
				while(y <= maxy && !this.yGap(y, minx, maxx)) y++;
				
				if(y - spany > 1) {
					var spanmaxy = y - 1;
					var spanx = minx;
					var x = minx;
					while(x <= maxx) {
						while(x <= maxx && !this.xGap(x, spany, spanmaxy)) x++;
						
						if(x - spanx > 1) this.addBand({x: spanx, y: spany}, {x: x - 1, y: spanmaxy});

						/* skip gap */
						while(x <= maxx && this.xGap(x, spany, spanmaxy)) x++;
						spanx = x;
					}
				}

				/* skip gap */
				while(y <= maxy && this.yGap(y, minx, maxx)) y++;
				spany = y;
			}
		}
	}

	VacuumWrap.prototype = {
		hourMap: [
			[7,  0,  1],
			[6, -1,  2],
			[5,  4,  3]
		],

		revHourMap: [
			{x:  0, y: -1},
			{x: +1, y: -1},
			{x: +1, y:  0},
			{x: +1, y: +1},
			{x:  0, y: +1},
			{x: -1, y: +1},
			{x: -1, y:  0},
			{x: -1, y: -1},
		],
		
		yGap: function(y, minx, maxx) {
			for(var x = minx; x <= maxx && !this.mapAt({x: x, y: y}); x++);
			return x > maxx;
		},
		
		xGap: function(x, miny, maxy) {
			for(var y = miny; y <= maxy && !this.mapAt({x: x, y: y}); y++);
			return y > maxy;
		},
		
		addBand: function(min, max) {
			var band = [];
			for(var i = min.x; i < max.x; i++) band.push({x: i, y: min.y});
			for(i = min.y; i < max.y; i++) band.push({x: max.x, y: i});
			for(i = max.x; i > min.x; i--) band.push({x: i, y: max.y});
			for(i = max.y; i > min.y; i--) band.push({x: min.x, y: i});

			var tmp = new RubberBand(band);
			this.bands.push({
				token: tmp,
				pivot: tmp.prev(),
				skipped: 0
			});
			
		},

		mapAt: function(p) {
			return p.x >= this.minx && p.x <= this.maxx && p.y >= this.miny && p.y <= this.maxy ?
				this.map[p.y - this.miny][p.x - this.minx] : false;
		},

		getHour: function(ref, node) {
			var xi = node.x - ref.x + 1;
			var yi = node.y - ref.y + 1;
			return this.hourMap[yi][xi];
		},

		isNeighbors: function(p1, p2) {
			return (Math.abs(p2.x - p1.x) <= 1) && (Math.abs(p2.y - p1.y) <= 1);
		},
		
		isDiag: function(p1, p2) {
			return (Math.abs(p2.x - p1.x) == 1) && (Math.abs(p2.y - p1.y) == 1);
		},
		
		getNthNeighbor: function(ref, num) {
			return {
				x: ref.x + this.revHourMap[num].x,
				y: ref.y + this.revHourMap[num].y
			};
		},

		canInflate: function(ref, node) {
			if(!this.isDiag(ref, node)) return false;
		
			var angle = this.getHour(ref, node);

			return this.mapAt(this.getNthNeighbor(ref, (angle - 1) & 0x7)) &&
					this.mapAt(this.getNthNeighbor(ref, (angle + 1) & 0x7));
		},

		/* scanline search */
		checkIntersect: function(band, newnode) {

			function ySort(n) {
				if(n[0].y > n[1].y) {
					return [n[0], n[1]];
				} else {
					return [n[1], n[0]];
				}
			}

			var check = [];
			if(this.isDiag(newnode.prev().val, newnode.val)) check.push(ySort([newnode.prev().val, newnode.val]));
			if(this.isDiag(newnode.val, newnode.next().val)) check.push(ySort([newnode.val, newnode.next().val]));

			if(band.find(function(node) {
				if(this.isDiag(node.prev().val, node.val)) {
					var p = ySort([node.prev().val, node.val]);
					
					return !!_.find(check, function(ck) {
						return ck[0].y == p[0].y && ck[0].x == p[1].x && p[0].x == ck[1].x;
					});
				}
			}, this)) return true;
			
			
			var tmp = [];
			band.each(function(obj) {
				dx = obj.val.x - obj.prev().val.x;
				dy = obj.val.y - obj.prev().val.y;

				tmp.push({
					x: obj.prev().val.x*2 + dx,
					y: obj.prev().val.y*2 + dy
				});
				
				tmp.push({
					x: obj.val.x*2,
					y: obj.val.y*2
				});
			});
			var view = _.sortBy(_.sortBy(new RubberBand(tmp).getAll(), function(o){return o.val.x;}), function(o){return o.val.y;});

			var i = 0;
			while(i < view.length) {
				
				var winding = 0;
				var y = view[i].val.y;

				while(i < view.length && view[i].val.y == y) {
					
					var x = view[i].val.x;
				
					while(i < view.length && view[i].val.x == x && view[i].val.y == y) {
						var inw = view[i].prev().val.y - view[i].val.y;
						var outw = view[i].val.y - view[i].next().val.y;
						
						winding += inw + outw;
						i++;
					}
					
					if(winding < 0) return true;
				}
			}

			return false;
		},

		bandDone: function(band) {
			return band.skipped && band.pivot == band.token.prev();
		},

		done: function() {
			var cnt = 0;
			_.each(this.bands, function(band) {
				cnt += Number(!this.bandDone(band));
			}, this);

			return !cnt;
		},

		step: function() {
			_.each(this.bands, function(band) {
				if(this.bandDone(band)) return;

				if(band.token.size() < 4) {
					/* reduce */
					this.bands.splice(this.bands.indexOf(band), 1);
					return;
				}
				
				/* maybe split */
				var opposite;
				if(opposite = band.token.find(function(node){
					return node != band.token &&
						node.val.x == band.token.val.x &&
						node.val.y == band.token.val.y;
				}, this)) {
					var tmp = [];
					while(opposite != band.token) {
						tmp.push(opposite.val);
						opposite = opposite.remove();
					}

					var bud = new RubberBand(tmp);
					this.bands.push({
						token: bud,
						pivot: bud.prev(),
						skipped: 0
					});
					
					band.pivot = band.token.prev();
					band.skipped = 0;
					return;
				}

				var inp = this.getHour(band.token.val, band.token.prev().val);
				var outp = this.getHour(band.token.val, band.token.next().val);
				var angle = (inp - outp) & 0x7;

				if(!this.mapAt(band.token.val)) {
					if(inp < 0 || outp < 0) {
						/* same node, reduce */
						band.pivot = band.token.prev();
						band.token = band.token.remove();
						band.skipped = 0;
						return;
					}

					if(angle == 0 || (angle < 3 &&
							(!this.isDiag(band.token.prev().val, band.token.val) ||
							!this.isDiag(band.token.val, band.token.next().val)))) {
						var clone = band.token.deepClone();
						var r = clone.remove();

						if(!this.checkIntersect(clone, r)) {
							/* in and out are neighbors */
							band.pivot = band.token.prev();
							band.token = band.token.remove();
							band.skipped = 0;
							return;
						}
					}

					/* Analyze inner neighbors */
					/* (maybe) move */
					for(i = 0; i < angle - 1; i++) {
					var nb = (outp + i + 1) & 0x7;
						var newNode = this.getNthNeighbor(band.token.val, nb);
					
						if(this.isNeighbors(newNode, band.token.prev().val) &&
								this.isNeighbors(newNode, band.token.next().val)) {

							var clone = band.token.deepClone();
							clone.val = newNode;

							if(!this.checkIntersect(clone, clone)) {
								/* can move there */
								band.token.val = newNode;
								band.pivot = band.token;
								band.token = band.token.next();
								band.skipped = 0;
								return;
							}
						}
					}
				}

				/* (maybe) insert nodes */
				/* check for in-node self intersection */
				if(!this.mapAt(band.token.val) ||
						!this.mapAt(band.token.prev().val)) {

					for(i = 0; i < angle - 1; i++) {
						var nb = (outp + i + 1) & 0x7;
						var newNode = this.getNthNeighbor(band.token.val, nb);
				
						/* check intersection */
						if(this.isNeighbors(newNode, band.token.prev().val) &&
								this.isNeighbors(newNode, band.token.val) &&

								!this.canInflate(band.token.prev().val, newNode) &&
								!this.canInflate(newNode, band.token.val)) {
								
							var clone = band.token.deepClone();
							var n = clone.before(newNode);

							if(!this.checkIntersect(clone, n)) {
								/* insert before */
								band.pivot = band.token.before(newNode);
								band.token = band.token.next();
								band.skipped = 0;
								return;
							}
						}
					}
				}

				band.token = band.token.next();
				band.skipped++;
			}, this);

			return this;
		},

		run: function() {
			while(!this.done()) this.step();
		},

		getData: function() {
			return _.map(this.bands, function(b){return b.token.getData();});
		}
	};

	game.App = function(options) {
		_.extend(this, _.pick(options, ["style", "xnodes", "ynodes"]));
		
		this.canvas = $('#board');
		
		this.canvasW = this.canvas.width() - this.style.board.padding * 2 - 1;
		this.gridStep = this.canvasW / (this.xnodes - 1);
		this.canvasH = this.gridStep * (this.ynodes - 1);
		
		this.canvas.attr("height", Math.round(this.canvasH + this.style.board.padding * 2 + 1));

		this.points = [];
		this.renderGame();

		var self = this;
		_.bindAll(this, "canvasClick", "runClick", "step");
		this.canvas.click(this.canvasClick);
		$("#run-btn").click(this.runClick);
		$("#step-btn").click(this.step);
	}

	game.App.prototype = {
		renderGame: function() {
			this.drawGrid();
			_.each(this.points, function(p) {
				this.drawPoint(p, this.style.playerB);
			}, this);
			
			this.drawBands();
			return this;
		},

		drawGrid: function() {
			var pad = this.style.board.padding;
			var ctx = this.canvas.get(0).getContext("2d");

			ctx.clearRect(0, 0, this.canvasW + pad * 2, this.canvasH + pad * 2);
			ctx.save();
			ctx.setStyle(this.style.board.grid);
			
			var i, x = 0, y = 0;
			for(i = 0; i < this.xnodes; i++) {
				ctx.beginPath();
				ctx.moveTo(pad + Math.round(x) + 0.5, pad + 0.5);
				ctx.lineTo(pad + Math.round(x) + 0.5, pad + this.canvasH + 0.5);
				ctx.stroke();
				x += this.gridStep;
			}
		
			for(i = 0; i < this.ynodes; i++) {
				ctx.beginPath();
				ctx.moveTo(pad + 0.5, pad + Math.round(y) + 0.5);
				ctx.lineTo(pad + this.canvasW + 0.5, pad + Math.round(y) + 0.5);
				ctx.stroke();
				y += this.gridStep;
			}
			ctx.restore();
			return this;
		},
		
		drawPoint: function(point, style) {
			var pad = this.style.board.padding;
			var ctx = this.canvas.get(0).getContext("2d");
			ctx.save();
			ctx.setStyle(style.point);
			ctx.beginPath();
			ctx.arc(pad + Math.round(point.x * this.gridStep) + 0.5,
					pad + Math.round(point.y * this.gridStep) + 0.5,
					style.radius, 0, Math.PI*2, true);
			ctx.fill();
			ctx.restore();

			return point;
		},

		drawBands: function() {
			var xs = this.gridStep;
			var ys = this.gridStep;
			var pad = this.style.board.padding;
		
			if(this.wrap) {
				var bands = this.wrap.getData();
				_.each(bands, function(band) {
					var ctx = this.canvas.get(0).getContext("2d");

					ctx.save();
					ctx.setStyle(this.style.playerB.area);

					ctx.beginPath();
					ctx.moveTo(
						pad + Math.round(band[0].x * xs) + 0.5,
						pad + Math.round(band[0].y * ys) + 0.5
					);
					_.each(band.slice(1), function(p) {
						ctx.lineTo(
							pad + Math.round(p.x * xs) + 0.5,
							pad + Math.round(p.y * ys) + 0.5
						);
					});
					ctx.closePath();
					ctx.fill();
					ctx.stroke();
					ctx.restore();

				}, this);
			}
			return this;
		},
		
		canvasClick: function(evt) {
			var offsetX, offsetY;
			
			if(evt.offsetX != undefined) {
				offsetX = evt.offsetX;
				offsetY = evt.offsetY;
			} else {
    			var rect = evt.target.getBoundingClientRect();
				offsetX = evt.clientX - rect.left;
          		offsetY = evt.clientY - rect.top;
			}
			
			var pad = this.style.board.padding;
			var xx = offsetX - pad;
			var yy = offsetY - pad;
			if(xx >= 0 && yy >= 0) {
				var xPos = Math.round(xx / this.gridStep);
				var yPos = Math.round(yy / this.gridStep);
				if(!_.find(this.points, function(p){return p.x == xPos && p.y == yPos;})) {
					this.points.push(this.drawPoint({x: xPos, y: yPos}, this.style.playerB));
				}
			}
		},

		runClick: function() {
			this.wrap = new VacuumWrap(this.points);
			this.wrap.run();
			this.renderGame();
		},

		step: function() {
			if(!this.wrap) this.wrap = new VacuumWrap(this.points);
			this.wrap.step();
			this.renderGame();
		}
	};
})();

$(document).ready(function(){
	window.app = new game.App({
		style: game.style,
		xnodes: 30,
		ynodes: 25,
	});
});
