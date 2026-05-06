# typed: true
# frozen_string_literal: true

class Privacies::ShowController < ApplicationController
  sig { returns(T.untyped) }
  def call
    redirect_to(Rails.configuration.mewst["privacy_url"], allow_other_host: true)
  end
end
