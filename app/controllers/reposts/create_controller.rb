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
    form = Forms::Repost.new(profile: current_profile!, post_id: params[:post_id])
    command = Commands::CreateRepost.new(form:)

    ActiveRecord::Base.transaction do
      command.call
    end

    head :created
  end
end
