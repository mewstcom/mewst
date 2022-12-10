# typed: true
# frozen_string_literal: true

class Posts::CreateController < ApplicationController
  include Authenticatable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    content, = post_creator_params.values_at(:content)
    @post_creator = T.must(current_profile).new_post(
      content:
    )

    if @post_creator.invalid?
      @posts = T.must(current_profile).home_timeline_posts
      return render("home/show/call", status: :unprocessable_entity)
    end

    @post_creator.call

    flash[:notice] = t("messages.posts.created")
    redirect_to home_path
  end

  private

  sig { returns(ActionController::Parameters) }
  def post_creator_params
    T.cast(params.require(:post_creator), ActionController::Parameters).permit(:content)
  end
end
