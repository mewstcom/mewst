# typed: true
# frozen_string_literal: true

class Posts::CreateController < ApplicationController
  include Authenticatable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = PostForm.new(form_params.merge(profile: T.must(current_user).profile))

    if @form.invalid?
      return render("home/show/call", status: :unprocessable_entity)
    end

    CreatePostService.new(form: @form).call

    flash[:notice] = t("messages.posts.created")
    redirect_to home_path
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:post_form), ActionController::Parameters).permit(:body)
  end
end
