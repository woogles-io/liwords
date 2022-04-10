This project was bootstrapped with [Create React App](https://github.com/facebook/create-react-app).

We are using `npm` globally.

### Notes

I mostly followed this page, and it was still a pain in the ass:

https://medium.com/@feralamillo/create-react-app-typescript-eslint-and-prettier-699277b0b913

New (Apr 2022):

Updating to CRA 5.0 was a nightmare. I deleted nearly every dev package from `devDependencies`, cleared out `node_modules` and `package-lock.json` then ran:

```
npm i -D --save-exact eslint-config-airbnb eslint-config-airbnb-typescript eslint-config-prettier eslint-import-resolver-typescript eslint-loader eslint-plugin-import eslint-plugin-jsx-a11y eslint-plugin-react eslint-plugin-react-hooks @typescript-eslint/parser @typescript-eslint/eslint-plugin

npm i -D --save-exact prettier prettier-eslint prettier-eslint-cli eslint-plugin-prettier
```

I think this is needed too:

```
npm i -D --save-exact sass
```

then `npm install` to fix things.
