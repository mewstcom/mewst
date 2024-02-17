# typed: true
# frozen_string_literal: true

class Stamps::CreateController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = StampForm.new(target_post_id: params[:post_id])

    respond_to do |format|
      if @form.invalid?
        return format.turbo_stream { render(status: :unprocessable_entity) }
      end

      result = CreateStampUseCase.new.call(current_actor: current_actor, target_post: @form.target_post.not_nil!)
      @post = result.post
      @stamp_checker = StampChecker.new(profile: current_actor!.profile, posts: [@post])

      format.turbo_stream
    end
  end
end
