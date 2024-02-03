# typed: true
# frozen_string_literal: true

class Posts::DestroyController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    form = PostDeleteForm.new(post_id: params[:post_id])
    form.validate!

    DeletePostUseCase.new(client: v1_public_client).call(form:)

    flash[:notice] = t("messages.posts.deleted")
    redirect_back(fallback_location: root_path)
  end
end
