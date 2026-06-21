# typed: true
# frozen_string_literal: true

class Posts::DestroyController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    form = DiscardPostForm.new(target_post_id: params[:post_id])
    form.profile = viewer!.profile_record.not_nil!

    # Treat a missing target post as a benign no-op instead of raising: it may be
    # already deleted (double submission, stale page) or not owned by the viewer,
    # none of which should surface to the user as a 500.
    #
    # [Ja] 対象の投稿が見つからない場合は例外を投げず no-op として扱う。既に削除済み
    # (二重送信・stale なページ) や閲覧者のものでない場合があり、いずれもユーザーに
    # 500 を見せるべきではないため。
    if form.invalid?
      return redirect_to(home_path, status: :see_other)
    end

    DiscardPostUseCase.new.call(target_post: form.target_post.not_nil!)

    flash[:notice] = t("messages.posts.deleted")
    redirect_to(home_path, status: :see_other)
  end
end
