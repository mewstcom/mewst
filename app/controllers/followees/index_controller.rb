# typed: true
# frozen_string_literal: true

class Followees::IndexController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    page = viewer!.followees.kept
      # NOTE: order で `follows.followed_at` を指定していると `page.records` の `id` が `follows.id` になってしまう
      #       https://github.com/healthie/activerecord_cursor_paginate/blob/0c262804e436964da9760f2ade1c66a9e3d2906b/lib/activerecord_cursor_paginate/paginator.rb#L84
      #       ↑で `Arel.star` が使われており、おそらく https://github.com/rails/rails/issues/41151 の影響で
      #       `profiles.id` が `follows.id` に上書きされてしまっていると思われる
      #       そのため `select("profiles.*")` を明示的に指定して `profiles` テーブルのカラムだけ選択するようにする
      .select("profiles.*")
      .cursor_paginate(
        after: params[:after].presence,
        before: params[:before].presence,
        limit: 15,
        order: {Arel.sql("follows.followed_at") => :desc, id: :desc}
      )
      .fetch
    @followees = page.records
    @page_info = PageInfo.from_cursor_paginate_page(page:)
    @follow_checker = FollowChecker.new(profile: viewer!.profile.not_nil!, target_profiles: @followees)
  end
end
