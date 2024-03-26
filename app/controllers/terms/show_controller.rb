# typed: true
# frozen_string_literal: true

class Terms::ShowController < ApplicationController
  sig { returns(T.untyped) }
  def call
    redirect_to(Rails.configuration.mewst["terms_url"], allow_other_host: true)
  end
end
