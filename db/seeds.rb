[
  ["me@shimba.co", "shimbaco", "password", "https://shimba.co/img/shimbaco.jpg"],
  ["user1@example.com", "user1", "password", ""],
  ["user2@example.com", "user2", "password", ""],
  ["user3@example.com", "user3", "password", ""]
].each do |(email, atname, password, avatar_url)|
  ActiveRecord::Base.transaction do
    verification = Verification.create!(email:, event: :sign_up, code: 111111, succeeded_at: Time.current)
    account_activation = AccountActivation.new(atname:, locale: "ja", password:)
    account_activation.verification = verification
    account = account_activation.run

    if avatar_url.present?
      profile = account.profiles.first
      profile.update!(avatar_url:)
    end
  end
end
