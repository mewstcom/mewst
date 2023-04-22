# typed: true
# frozen_string_literal: true

class Reposts::CreateController < ApplicationController
  include Authenticatable
  include Authorizable
  include Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @repost_creator = Repost::Creator.new(form_params)
    @repost_creator.profile = current_profile!

    ActiveRecord::Base.transaction do
      @repost_creator.call
    end

    head :created
  end

  private

  sig { returns(ActionController::Parameters) }
  def form_params
    T.cast(params.require(:repost_creator), ActionController::Parameters).permit(:post_id)
  end
end
