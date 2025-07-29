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
    form.validate!

    DiscardPostUseCase.new.call(target_post: form.target_post.not_nil!)

    flash[:notice] = t("messages.posts.deleted")
    redirect_to(home_path, status: :see_other)
  end
end
