# typed: true
# frozen_string_literal: true

class Communities::ShowController < ApplicationController
  sig { returns(T.untyped) }
  def call
    redirect_to(Rails.configuration.mewst["community_url"], allow_other_host: true)
  end
end
